package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func Respond(ctx context.Context, w http.ResponseWriter, statusCode int, data any) error {

	//set response status code
	setResponseStatus(ctx, statusCode)

	//check to see if client cancelled the request
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errors.New("client cancelled the request")
		}
	}

	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("encode to JSON: %w", err)
	}

	return nil
}
