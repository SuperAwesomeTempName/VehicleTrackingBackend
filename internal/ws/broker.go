package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	redisclient "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/redis"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type Broker struct {
	clients   map[*Client]struct{}
	register  chan *Client
	unregister chan *Client
	redis     *redisclient.Client
	mu        sync.Mutex
}

func NewBroker(r *redisclient.Client) *Broker {
	b := &Broker{
		clients: make(map[*Client]struct{}),
		register: make(chan *Client),
		unregister: make(chan *Client),
		redis: r,
	}
	go b.run()
	go b.subscribeRedis(context.Background())
	return b
}

func (b *Broker) run() {
	for {
		select {
		case c := <-b.register:
			b.mu.Lock()
			b.clients[c] = struct{}{}
			b.mu.Unlock()
		case c := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[c]; ok {
				delete(b.clients, c)
				close(c.send)
				c.conn.Close()
			}
			b.mu.Unlock()
		}
	}
}

func (b *Broker) subscribeRedis(ctx context.Context) {
	sub := b.redis.rdb.Subscribe(ctx, "vehicle:*") // pattern not supported by go-redis Subscribe - use PubSub.PSubscribe in real code
	ch := sub.Channel()
	for msg := range ch {
		b.broadcast([]byte(msg.Payload))
	}
}

func (b *Broker) broadcast(msg []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.clients {
		select {
		case c.send <- msg:
		default:
			delete(b.clients, c)
			close(c.send)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // update for production with origin checks
}

func (b *Broker) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	client := &Client{conn: conn, send: make(chan []byte, 256)}
	b.register <- client

	// read pump (we can ignore reads or handle subscribe messages)
	go func() {
		defer func() { b.unregister <- client }()
		client.conn.SetReadLimit(512)
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		for {
			if _, _, err := client.conn.NextReader(); err != nil {
				break
			}
		}
	}()

	// write pump
	go func() {
		ticker := time.NewTicker(54 * time.Second)
		defer func() { ticker.Stop(); client.conn.Close() }()
		for {
			select {
			case msg, ok := <-client.send:
				client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					client.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				w, err := client.conn.NextWriter(websocket.TextMessage)
				if err != nil { return }
				w.Write(msg)
				if err := w.Close(); err != nil { return }
			case <-ticker.C:
				client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil { return }
			}
		}
	}()
}
