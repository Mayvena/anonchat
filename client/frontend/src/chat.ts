import { createId } from './id';

export type ChatConnectionState =
  | 'connecting'
  | 'connected'
  | 'degraded'
  | 'reconnecting'
  | 'failed';

export type SendHandler = (to: string, body: string) => boolean | Promise<boolean>;

const STATE_LABEL: Record<ChatConnectionState, string> = {
  connecting:   'CONNECTING',
  connected:    'CONNECTED',
  degraded:     'DEGRADED',
  reconnecting: 'RECONNECTING',
  failed:       'FAILED',
};

export interface ChatUser {
  id: string;
  displayName: string;
}

export interface ChatMessageInput {
  id?: string;
  fromUserId: string;
  text: string;
  ttlSeconds: number;
}

interface MessageRuntime {
  id: string;
  element: HTMLElement;
  timerElement: HTMLElement;
  expiresAtMs: number;
  intervalId: number;
}

export class ChatPanel {
  private readonly usersEl: HTMLElement;
  private readonly messagesEl: HTMLElement;
  private readonly muteBtn: HTMLButtonElement;
  private readonly users = new Map<string, ChatUser>();
  private readonly messages = new Map<string, MessageRuntime>();
  private readonly audioCtx: AudioContext;
  private muted = false;
  private readonly statusEl: HTMLElement;
  private readonly formEl: HTMLFormElement;
  private readonly toInput: HTMLInputElement;
  private readonly bodyInput: HTMLTextAreaElement;
  private readonly sendBtn: HTMLButtonElement;
  private readonly retryBtn: HTMLButtonElement;
  private sendHandler: SendHandler | null = null;
  private retryHandler: (() => void) | null = null;

  constructor() {
    this.usersEl = this.requireEl('chat-users');
    this.messagesEl = this.requireEl('chat-messages');
    this.muteBtn = this.requireEl('sound-toggle') as HTMLButtonElement;
    this.statusEl = this.requireEl('chat-conn-status');
    this.formEl = this.requireEl('chat-form') as HTMLFormElement;
    this.toInput = this.requireEl('chat-to') as HTMLInputElement;
    this.bodyInput = this.requireEl('chat-body') as HTMLTextAreaElement;
    this.sendBtn = this.requireEl('chat-send') as HTMLButtonElement;
    this.retryBtn = this.requireEl('chat-retry') as HTMLButtonElement;

    this.formEl.addEventListener('submit', (e) => {
      e.preventDefault();
      this.submit();
    });
    // Enter sends, Shift+Enter inserts a newline.
    this.bodyInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
        e.preventDefault();
        this.submit();
      }
    });
    this.retryBtn.addEventListener('click', () => this.retryHandler?.());
    this.setConnectionState('connecting');

    const AudioCtor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    this.audioCtx = new AudioCtor();

    this.muteBtn.addEventListener('click', () => this.toggleMute());
    this.renderMuteButton();
  }

  onSend(handler: SendHandler): void {
    this.sendHandler = handler;
  }

  onRetry(handler: () => void): void {
    this.retryHandler = handler;
  }

  hasUser(id: string): boolean {
    return this.users.has(id);
  }

  setConnectionState(state: ChatConnectionState): void {
    this.statusEl.textContent = STATE_LABEL[state];
    this.statusEl.dataset.state = state;
    const canSend = state === 'connected' || state === 'degraded';
    this.sendBtn.disabled = !canSend;
    this.retryBtn.hidden = state !== 'failed';
  }

  private submit(): void {
    const to = this.toInput.value.trim();
    const body = this.bodyInput.value.trim();
    if (!to || !body || !this.sendHandler) return;

    void (async () => {
      const ok = await this.sendHandler?.(to, body);
      if (ok) {
        this.bodyInput.value = '';
      }
      this.bodyInput.focus();
    })();
  }

  setUsers(users: ChatUser[]): void {
    this.users.clear();
    users.forEach((u) => this.users.set(u.id, u));
    this.renderUsers();
  }

  addUser(user: ChatUser): void {
    this.users.set(user.id, user);
    this.renderUsers();
  }

  addMessage(input: ChatMessageInput): void {
    const id = input.id ?? createId();
    const fromName = this.users.get(input.fromUserId)?.displayName ?? input.fromUserId;

    const msgEl = document.createElement('article');
    msgEl.className = 'chat-message';
    msgEl.dataset.messageId = id;

    const header = document.createElement('header');
    header.className = 'chat-message-header';

    const fromEl = document.createElement('span');
    fromEl.className = 'chat-from';
    fromEl.textContent = fromName;

    const timerEl = document.createElement('span');
    timerEl.className = 'chat-timer';
    timerEl.textContent = `⏳ ${input.ttlSeconds}s`;

    header.append(fromEl, timerEl);

    const body = document.createElement('p');
    body.className = 'chat-text';
    body.textContent = input.text;

    msgEl.append(header, body);
    this.messagesEl.appendChild(msgEl);
    this.messagesEl.scrollTop = this.messagesEl.scrollHeight;

    const expiresAtMs = Date.now() + input.ttlSeconds * 1000;
    const intervalId = window.setInterval(() => {
      const remaining = Math.max(0, Math.ceil((expiresAtMs - Date.now()) / 1000));
      timerEl.textContent = `⏳ ${remaining}s`;
      if (remaining === 0) {
        this.removeMessage(id);
      }
    }, 250);

    this.messages.set(id, {
      id,
      element: msgEl,
      timerElement: timerEl,
      expiresAtMs,
      intervalId,
    });
  }

  private removeMessage(id: string): void {
    const runtime = this.messages.get(id);
    if (!runtime) return;

    window.clearInterval(runtime.intervalId);
    runtime.element.classList.add('expiring');

    window.setTimeout(() => {
      runtime.element.remove();
    }, 220);

    this.messages.delete(id);
    this.playDeleteSound();
  }

  private renderUsers(): void {
    this.usersEl.textContent = '';

    const fragment = document.createDocumentFragment();
    for (const user of this.users.values()) {
      const item = document.createElement('div');
      item.className = 'chat-user-item';

      const dot = document.createElement('span');
      dot.className = 'chat-user-dot';

      const name = document.createElement('span');
      name.className = 'chat-user-name';
      name.textContent = user.displayName;

      item.append(dot, name);
      fragment.appendChild(item);
    }

    this.usersEl.appendChild(fragment);
  }

  private toggleMute(): void {
    this.muted = !this.muted;
    this.renderMuteButton();
  }

  private renderMuteButton(): void {
    this.muteBtn.textContent = this.muted ? '🔇' : '🔊';
    this.muteBtn.setAttribute('aria-label', this.muted ? 'Unmute system sounds' : 'Mute system sounds');
    this.muteBtn.title = this.muted ? 'Unmute' : 'Mute';
  }

  private async playDeleteSound(): Promise<void> {
    if (this.muted) return;

    if (this.audioCtx.state === 'suspended') {
      await this.audioCtx.resume();
    }

    const now = this.audioCtx.currentTime;

    const osc = this.audioCtx.createOscillator();
    osc.type = 'triangle';
    osc.frequency.setValueAtTime(1040, now);
    osc.frequency.exponentialRampToValueAtTime(220, now + 0.12);

    const gain = this.audioCtx.createGain();
    gain.gain.setValueAtTime(0.0001, now);
    gain.gain.exponentialRampToValueAtTime(0.12, now + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.14);

    osc.connect(gain);
    gain.connect(this.audioCtx.destination);

    osc.start(now);
    osc.stop(now + 0.16);
  }

  private requireEl(id: string): HTMLElement {
    const el = document.getElementById(id);
    if (!el) {
      throw new Error(`Missing required element: #${id}`);
    }
    return el;
  }
}
