package syncbox

type queue struct {
	files []File
}

func NewQueue() *queue {
	return &queue{}
}

func (q *queue) push(files ...File) {
	q.files = append(q.files, files...)
}

func (q *queue) pop() File {
	file := q.files[0]
	q.files = q.files[1:]
	return file
}
