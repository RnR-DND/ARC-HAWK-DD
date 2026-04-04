import { useEffect, useRef, useState, useCallback } from 'react';

// WebSocket message types (matching backend)
export type WSMessageType =
  | 'scan_progress'
  | 'new_finding'
  | 'scan_complete'
  | 'system_status'
  | 'scan_started';

export interface WSMessage {
  type: WSMessageType;
  data: any;
  timestamp: string;
}

interface UseWebSocketOptions {
  url?: string;
  onMessage?: (message: WSMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface UseWebSocketReturn {
  socket: WebSocket | null;
  isConnected: boolean;
  isConnecting: boolean;
  connect: () => void;
  disconnect: () => void;
  send: (message: any) => void;
  lastMessage: WSMessage | null;
}

export function useWebSocket(options: UseWebSocketOptions = {}): UseWebSocketReturn {
  const {
    url = `ws://localhost:8080/api/v1/ws`,
    onMessage,
    onConnect,
    onDisconnect,
    onError,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5
  } = options;

  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);

  const reconnectAttempts = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const socketRef = useRef<WebSocket>();

  const connect = useCallback(() => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setIsConnecting(true);

    try {
      const ws = new WebSocket(url);
      socketRef.current = ws;
      setSocket(ws);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setIsConnecting(false);
        reconnectAttempts.current = 0;
        onConnect?.();
      };

      ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          setLastMessage(message);
          onMessage?.(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        setIsConnected(false);
        setIsConnecting(false);
        setSocket(null);
        socketRef.current = undefined;
        onDisconnect?.();

        // Attempt to reconnect with exponential backoff
        if (event.code !== 1000 && reconnectAttempts.current < maxReconnectAttempts) {
          reconnectAttempts.current++;
          const backoff = Math.min(
            reconnectInterval * Math.pow(2, reconnectAttempts.current - 1),
            30000
          );
          console.log(`Reconnecting in ${backoff}ms (${reconnectAttempts.current}/${maxReconnectAttempts})...`);

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, backoff);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        onError?.(error);
      };

    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      setIsConnecting(false);
    }
  }, [url, onMessage, onConnect, onDisconnect, onError, reconnectInterval, maxReconnectAttempts]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = undefined;
    }

    if (socketRef.current) {
      socketRef.current.close(1000, 'Client disconnect');
      socketRef.current = undefined;
    }

    setSocket(null);
    setIsConnected(false);
    setIsConnecting(false);
  }, []);

  const send = useCallback((message: any) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected. Cannot send message:', message);
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return {
    socket,
    isConnected,
    isConnecting,
    connect,
    disconnect,
    send,
    lastMessage
  };
}

// Hook for real-time scan monitoring
export function useScanMonitoring(scanId?: string) {
  const [scanProgress, setScanProgress] = useState<{
    progress: number;
    status: string;
    message: string;
    findings: any[];
  }>({
    progress: 0,
    status: 'idle',
    message: '',
    findings: []
  });

  const { connect, disconnect, isConnected } = useWebSocket({
    onMessage: (message) => {
      switch (message.type) {
        case 'scan_started':
          if (!scanId || message.data.scan_id === scanId) {
            setScanProgress(prev => ({
              ...prev,
              progress: 0,
              status: 'running',
              message: `Started scanning ${message.data.source}`
            }));
          }
          break;

        case 'scan_progress':
          if (!scanId || message.data.scan_id === scanId) {
            setScanProgress(prev => ({
              ...prev,
              progress: message.data.progress,
              status: message.data.status,
              message: message.data.message
            }));
          }
          break;

        case 'new_finding':
          if (!scanId || message.data.scan_id === scanId) {
            setScanProgress(prev => ({
              ...prev,
              findings: [...prev.findings, message.data]
            }));
          }
          break;

        case 'scan_complete':
          if (!scanId || message.data.scan_id === scanId) {
            setScanProgress(prev => ({
              ...prev,
              progress: 100,
              status: 'completed',
              message: `Scan completed with ${message.data.total_findings} findings`
            }));
          }
          break;
      }
    }
  });

  useEffect(() => {
    connect();
    return () => disconnect();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Empty deps - connect/disconnect are stable via useCallback

  return {
    scanProgress,
    isConnected
  };
}

// Hook for system status monitoring
export function useSystemStatus() {
  const [systemStatus, setSystemStatus] = useState<{
    connectedClients: number;
    activeScans: number;
    lastUpdate: string;
  }>({
    connectedClients: 0,
    activeScans: 0,
    lastUpdate: new Date().toISOString()
  });

  const { connect, disconnect } = useWebSocket({
    onMessage: (message) => {
      if (message.type === 'system_status') {
        setSystemStatus({
          connectedClients: message.data.connected_clients || 0,
          activeScans: message.data.active_scans || 0,
          lastUpdate: message.timestamp
        });
      }
    }
  });

  useEffect(() => {
    connect();
    return () => disconnect();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Empty deps - connect/disconnect are stable via useCallback

  return systemStatus;
}