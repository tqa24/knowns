/**
 * OpenCodeEventContext — singleton global EventSource for OpenCode events.
 *
 * Maintains exactly ONE SSE connection to /api/opencode/global/event regardless
 * of how many components subscribe. Subscribers call `subscribe(handler)` and
 * receive a cleanup function. This prevents HTTP/1.1 connection slot exhaustion
 * caused by per-component EventSource instances.
 */

import { createContext, useCallback, useContext, useEffect, useRef, type ReactNode } from "react";
import { opencodeApi } from "../api/client";
import { useOpenCode } from "./OpenCodeContext";

type EventHandler = (data: unknown) => void;
type Unsubscribe = () => void;

interface OpenCodeEventContextType {
	subscribe: (handler: EventHandler) => Unsubscribe;
}

const OpenCodeEventContext = createContext<OpenCodeEventContextType | undefined>(undefined);

export function OpenCodeEventProvider({ children }: { children: ReactNode }) {
	const { status } = useOpenCode();
	const handlersRef = useRef<Set<EventHandler>>(new Set());
	const esRef = useRef<EventSource | null>(null);
	const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	const dispatch = useCallback((data: unknown) => {
		handlersRef.current.forEach((h) => {
			try {
				h(data);
			} catch {
				// ignore handler errors
			}
		});
	}, []);

	useEffect(() => {
		if (!status?.available) {
			// Clean up any open connection while OpenCode is offline.
			esRef.current?.close();
			esRef.current = null;
			return;
		}

		let cancelled = false;

		const connect = () => {
			if (cancelled) return;
			try {
				const es = opencodeApi.eventSource();
				esRef.current = es;

				es.onmessage = (event) => {
					try {
						const rawData = JSON.parse(event.data as string);
						const data = rawData?.payload ?? rawData;
						dispatch(data);
					} catch {
						// ignore parse errors
					}
				};

				es.addEventListener("question.asked", (namedEvent: MessageEvent) => {
					try {
						const rawData = JSON.parse(namedEvent.data as string);
						dispatch(rawData?.payload ?? rawData);
					} catch {
						// ignore
					}
				});

				es.onerror = () => {
					es.close();
					esRef.current = null;
					if (!cancelled) {
						reconnectTimerRef.current = setTimeout(connect, 3000);
					}
				};
			} catch {
				if (!cancelled) {
					reconnectTimerRef.current = setTimeout(connect, 3000);
				}
			}
		};

		connect();

		return () => {
			cancelled = true;
			if (reconnectTimerRef.current) {
				clearTimeout(reconnectTimerRef.current);
				reconnectTimerRef.current = null;
			}
			esRef.current?.close();
			esRef.current = null;
		};
	}, [status?.available, dispatch]);

	const subscribe = useCallback((handler: EventHandler): Unsubscribe => {
		handlersRef.current.add(handler);
		return () => {
			handlersRef.current.delete(handler);
		};
	}, []);

	return (
		<OpenCodeEventContext.Provider value={{ subscribe }}>
			{children}
		</OpenCodeEventContext.Provider>
	);
}

export function useOpenCodeEvent() {
	const ctx = useContext(OpenCodeEventContext);
	if (!ctx) {
		throw new Error("useOpenCodeEvent must be used within OpenCodeEventProvider");
	}
	return ctx;
}
