package syncbox

type Message struct {
	Command string `json:"cmd"`
	Files   []File `json:"files"`
}
