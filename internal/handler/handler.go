package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"warehouse/internal/model"
	"warehouse/internal/service"
)

// Server 是 HTTP 接口层。
type Server struct {
	svc *service.Service
}

func New(svc *service.Service) *Server {
	return &Server{svc: svc}
}

// Routes 返回路由表。
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /orders", s.listOrders)
	mux.HandleFunc("GET /orders/{id}", s.getOrder)
	mux.HandleFunc("POST /orders", s.createOrder)
	return mux
}

type createRequest struct {
	SKUs     []string `json:"skus"`
	Priority int      `json:"priority"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	var status *model.Status
	if raw := r.URL.Query().Get("status"); raw != "" {
		st := model.Status(raw)
		status = &st
	}
	orders, err := s.svc.ListOrders(status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list orders failed")
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := s.svc.FindOrder(id)
	if err != nil {
		s.writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	o, err := s.svc.CreateOrder(req.SKUs, model.Priority(req.Priority))
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "create order failed")
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) writeOrderError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrOrderNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if errors.Is(err, service.ErrInvalidTransition) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, service.ErrInsufficientStock) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}
