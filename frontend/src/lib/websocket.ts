type EventCallback = (data: unknown) => void;

export class WSClient {
  private ws: WebSocket | null = null;
  private listeners = new Map<string, Set<EventCallback>>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private maxDelay = 30000;
  private token: string;
  private disposed = false;
  private onStatusChange?: (connected: boolean, reconnecting: boolean) => void;

  constructor(token: string, onStatusChange?: (connected: boolean, reconnecting: boolean) => void) {
    this.token = token;
    this.onStatusChange = onStatusChange;
    this.connect();
  }

  private connect() {
    if (this.disposed) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.ws = new WebSocket(`${protocol}//${host}/api/ws?token=${this.token}`);

    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.onStatusChange?.(true, false);
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        const cbs = this.listeners.get(msg.type);
        if (cbs) cbs.forEach((cb) => cb(msg.data));
      } catch { /* ignore */ }
    };

    this.ws.onclose = () => {
      this.onStatusChange?.(false, true);
      this.scheduleReconnect();
    };

    this.ws.onerror = () => { this.ws?.close(); };
  }

  private scheduleReconnect() {
    if (this.disposed) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxDelay);
      this.connect();
    }, this.reconnectDelay);
  }

  on(event: string, callback: EventCallback) {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event)!.add(callback);
    return () => { this.listeners.get(event)?.delete(callback); };
  }

  disconnect() {
    this.disposed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.listeners.clear();
    this.onStatusChange?.(false, false);
  }
}
