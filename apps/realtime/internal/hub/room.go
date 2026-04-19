package hub

import "log/slog"

// Room represents a group of clients connected to the same session.
type Room struct {
	Code    string
	clients map[*Client]bool
}

func newRoom(code string) *Room {
	return &Room{
		Code:    code,
		clients: make(map[*Client]bool),
	}
}

func (r *Room) add(c *Client) {
	r.clients[c] = true
	slog.Info("client joined room", "code", r.Code, "clients", len(r.clients))
}

func (r *Room) remove(c *Client) {
	if _, ok := r.clients[c]; ok {
		delete(r.clients, c)
		slog.Info("client left room", "code", r.Code, "clients", len(r.clients))
	}
}

func (r *Room) broadcast(message []byte) {
	for c := range r.clients {
		select {
		case c.send <- message:
		default:
			// Client send buffer full — drop the client
			close(c.send)
			delete(r.clients, c)
		}
	}
}

func (r *Room) isEmpty() bool {
	return len(r.clients) == 0
}

func (r *Room) count() int {
	return len(r.clients)
}
