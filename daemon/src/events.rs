use tokio::sync::broadcast;

use crate::models::SseEvent;

/// Default channel capacity for SSE events.
const EVENT_CHANNEL_CAPACITY: usize = 256;

/// Event bus for broadcasting real-time updates to SSE clients.
#[derive(Clone)]
pub struct EventBus {
    sender: broadcast::Sender<SseEvent>,
}

impl EventBus {
    /// Create a new event bus.
    pub fn new() -> Self {
        let (sender, _) = broadcast::channel(EVENT_CHANNEL_CAPACITY);
        Self { sender }
    }

    /// Publish an event to all subscribers.
    pub fn publish(&self, event: SseEvent) {
        // Ignore send errors (no subscribers)
        let _ = self.sender.send(event);
    }

    /// Subscribe to receive events.
    pub fn subscribe(&self) -> broadcast::Receiver<SseEvent> {
        self.sender.subscribe()
    }
}

impl Default for EventBus {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[tokio::test]
    async fn test_event_bus_publish_subscribe() {
        let bus = EventBus::new();
        let mut receiver = bus.subscribe();

        let event = SseEvent {
            event_type: "test".to_string(),
            workspace_id: Some("ws-1".to_string()),
            data: json!({"message": "hello"}),
        };

        bus.publish(event.clone());

        let received = receiver.recv().await.unwrap();
        assert_eq!(received.event_type, "test");
        assert_eq!(received.workspace_id, Some("ws-1".to_string()));
    }

    #[tokio::test]
    async fn test_event_bus_no_subscribers() {
        let bus = EventBus::new();
        // Should not panic even with no subscribers
        bus.publish(SseEvent {
            event_type: "test".to_string(),
            workspace_id: None,
            data: json!(null),
        });
    }

    #[tokio::test]
    async fn test_event_bus_multiple_subscribers() {
        let bus = EventBus::new();
        let mut rx1 = bus.subscribe();
        let mut rx2 = bus.subscribe();

        bus.publish(SseEvent {
            event_type: "update".to_string(),
            workspace_id: None,
            data: json!({"key": "value"}),
        });

        let e1 = rx1.recv().await.unwrap();
        let e2 = rx2.recv().await.unwrap();
        assert_eq!(e1.event_type, "update");
        assert_eq!(e2.event_type, "update");
    }
}
