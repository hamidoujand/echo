package web

type Middleware func(HandlerFunc) HandlerFunc

func applyMiddleware(h HandlerFunc, mids ...Middleware) HandlerFunc {
	for i := len(mids) - 1; i >= 0; i-- {
		mid := mids[i]
		if mid != nil {
			h = mid(h)
		}
	}

	return h
}
