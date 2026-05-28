package http

// createQueueReq is the inbound DTO for POST /queues.
type createQueueReq struct {
	Name     string `json:"name" binding:"required"`
	Capacity int    `json:"capacity" binding:"required"`
}

type createQueueResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type pushReq struct {
	Value string `json:"value"`
	TTL   int    `json:"ttl"` // seconds; 0 = never expires
}

type messageResp struct {
	ID        string `json:"id"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
}

type statusResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int    `json:"size"`
	Capacity int    `json:"capacity"`
}

type queueListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type errorResp struct {
	Error string `json:"error"`
}
