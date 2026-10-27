/**
 * ws.ts – Chat router websocket client.
 *
 * Speaks the router envelopes (`session`, `message`) on `/ws`, retaining the
 * server-issued identity and credential in memory for authenticated reconnects.
 */

export type ConnectionState =
  | 'connecting'
  | 'connected'
  | 'degraded'
  | 'reconnecting'
  | 'failed';

export interface IncomingMessage {
  id:     string;
  from:   string;
  to:     string;
  body:   string;
  sentAt: number;
}

export interface RouterClientOptions {
  url:             string;
  onState:         (state: ConnectionState) => void;
  onSession:       (userId: string) => void;
  onMessage:       (msg: IncomingMessage) => void;
  maxAttempts?:    number;
  baseDelayMs?:    number;
  maxDelayMs?:     number;
}

const MAX_BODY_CHARS = 8192;
const DEGRADED_BUFFER_BYTES = 64 * 1024;

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null;
}

/** Returns null for anything that is not a well-formed `message` envelope. */
export function parseIncoming(raw: unknown): IncomingMessage | null {
  if (typeof raw !== 'string') return null;

  let env: unknown;
  try {
    env = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(env) || env.type !== 'message' || !isRecord(env.payload)) return null;

  const p = env.payload;
  if (
    typeof p.id !== 'string' ||
    typeof p.from !== 'string' ||
    typeof p.to !== 'string' ||
    typeof p.body !== 'string'
  ) {
    return null;
  }

  return {
    id:     p.id,
    from:   p.from,
    to:     p.to,
    body:   p.body,
    sentAt: typeof p.sent_at === 'number' ? p.sent_at : Math.floor(Date.now() / 1000),
  };
}

export class RouterClient {
  private socket: WebSocket | null = null;
  private attempts = 0;
  private retryTimer: number | null = null;
  private closedByUser = false;
  private state: ConnectionState = 'connecting';
  private userId: string | null = null;
  private credential: string | null = null;

  private readonly maxAttempts: number;
  private readonly baseDelayMs: number;
  private readonly maxDelayMs:  number;

  constructor(private readonly opts: RouterClientOptions) {
    this.maxAttempts = opts.maxAttempts ?? 8;
    this.baseDelayMs = opts.baseDelayMs ?? 500;
    this.maxDelayMs  = opts.maxDelayMs  ?? 15_000;
  }

  connect(): void {
    this.closedByUser = false;
    this.clearRetry();
    this.setState(this.attempts === 0 ? 'connecting' : 'reconnecting');

    const protocols = this.userId && this.credential
      ? [`anonchat.v1.auth.${this.userId}.${this.credential}`]
      : undefined;
    let socket: WebSocket;
    try {
      socket = protocols ? new WebSocket(this.opts.url, protocols) : new WebSocket(this.opts.url);
    } catch {
      this.scheduleReconnect();
      return;
    }
    this.socket = socket;

    socket.addEventListener('message', (ev: MessageEvent) => {
      if (typeof ev.data !== 'string') return;
      const session = this.parseSession(ev.data);
      if (session) {
        this.userId = session.userId;
        this.credential = session.credential;
        this.opts.onSession(session.userId);
        this.attempts = 0;
        this.setState('connected');
        return;
      }
      const msg = parseIncoming(ev.data);
      if (msg) this.opts.onMessage(msg);
    });

    socket.addEventListener('close', () => {
      if (this.socket === socket) this.socket = null;
      if (!this.closedByUser) this.scheduleReconnect();
    });

    // `close` always follows `error`; reconnect is handled there.
    socket.addEventListener('error', () => undefined);
  }

  /** Returns false when the message could not be handed to the socket. */
  send(id: string, to: string, body: string): boolean {
    const recipient = to.trim();
    if (!recipient || !body || body.length > MAX_BODY_CHARS) return false;

    const s = this.socket;
    if (!s || s.readyState !== WebSocket.OPEN) {
      if (this.state === 'connected') this.setState('degraded');
      return false;
    }

    s.send(JSON.stringify({ id, to: recipient, body }));
    this.setState(s.bufferedAmount > DEGRADED_BUFFER_BYTES ? 'degraded' : 'connected');
    return true;
  }

  close(): void {
    this.closedByUser = true;
    this.clearRetry();
    this.socket?.close();
    this.socket = null;
  }

  /** Manual retry after the `failed` state. */
  retry(): void {
    if (this.state !== 'failed') return;
    this.attempts = 0;
    this.connect();
  }

  private parseSession(raw: string): { userId: string; credential: string } | null {
    try {
      const env: unknown = JSON.parse(raw);
      if (!isRecord(env) || env.type !== 'session' || !isRecord(env.payload)) return null;
      const { user_id: userId, credential } = env.payload;
      if (typeof userId !== 'string' || !userId || typeof credential !== 'string' || !credential) return null;
      return { userId, credential };
    } catch {
      return null;
    }
  }

  private scheduleReconnect(): void {
    if (this.attempts >= this.maxAttempts) {
      this.setState('failed');
      return;
    }

    const delay = Math.min(this.maxDelayMs, this.baseDelayMs * 2 ** this.attempts);
    this.attempts++;
    this.setState('reconnecting');
    this.retryTimer = window.setTimeout(() => this.connect(), delay);
  }

  private clearRetry(): void {
    if (this.retryTimer !== null) {
      window.clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }

  private setState(next: ConnectionState): void {
    if (next === this.state) return;
    this.state = next;
    this.opts.onState(next);
  }
}

export { MAX_BODY_CHARS };
