/**
 * terminal.ts – Live streaming log panel with colour-coded severity levels.
 *
 * Appends structured log lines from the Rust core in real-time.
 * Auto-scrolls to the newest entry; prunes oldest when MAX_LINES is reached.
 * All user-supplied strings are HTML-escaped before insertion.
 */

export type LogLevel = 'info' | 'warn' | 'error';

export interface LogEntry {
  level:       LogLevel;
  message:     string;
  timestampMs: number;
}

const MAX_LINES = 300;

const LEVEL_COLOR: Record<LogLevel, string> = {
  info:  '#00d4ff',
  warn:  '#ffb700',
  error: '#ff4444',
};

const LEVEL_BADGE: Record<LogLevel, string> = {
  info:  'INFO ',
  warn:  'WARN ',
  error: 'ERROR',
};

export class TerminalPanel {
  private lineCount = 0;

  constructor(private readonly el: HTMLElement) {
    // Base container styles are applied via CSS class in index.html,
    // but we enforce the critical layout properties here as a safety net.
    el.style.overflowY    = 'auto';
    el.style.overflowX    = 'hidden';
    el.style.scrollBehavior = 'auto';   // instant scroll, not smooth (avoids lag)
  }

  append(entry: LogEntry): void {
    // Prune oldest line to keep DOM lean.
    if (this.lineCount >= MAX_LINES) {
      this.el.firstChild?.remove();
      this.lineCount--;
    }

    const line = document.createElement('div');
    line.className = 'log-line';

    const ts    = this.formatTs(entry.timestampMs);
    const color = LEVEL_COLOR[entry.level]  ?? '#c0c0c0';
    const badge = LEVEL_BADGE[entry.level]  ?? 'LOG  ';

    // Each line: timestamp · badge · message – all in separate spans so CSS
    // can target them independently.
    line.innerHTML =
      `<span class="log-ts">${ts}</span>` +
      `<span class="log-badge" style="color:${color}">${badge}</span>` +
      `<span class="log-msg">${escapeHtml(entry.message)}</span>`;

    this.el.appendChild(line);
    this.lineCount++;

    // Only scroll if the user is already near the bottom (within 80 px).
    const atBottom =
      this.el.scrollHeight - this.el.scrollTop - this.el.clientHeight < 80;
    if (atBottom) {
      this.el.scrollTop = this.el.scrollHeight;
    }
  }

  clear(): void {
    this.el.textContent = '';
    this.lineCount = 0;
  }

  private formatTs(ms: number): string {
    const d  = new Date(ms);
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    const ss = String(d.getSeconds()).padStart(2, '0');
    const ms3 = String(d.getMilliseconds()).padStart(3, '0');
    return `${hh}:${mm}:${ss}.${ms3}`;
  }
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
