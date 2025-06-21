package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (handler *HttpHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	orderID := chi.URLParam(r, "orderID")

	if orderID == "" {

		// --- Parse Query Parameters ---
		query := r.URL.Query()

		// Filters
		filter := make(map[string]interface{})

		filter["user_id"] = id

		if product := query.Get("product"); product != "" {
			filter["product"] = product
		}
		if search := query.Get("search"); search != "" {
			filter["search"] = search
		}
		if status := query.Get("status"); status != "" {
			filter["status"] = status
		}
		if start := query.Get("start_date"); start != "" {
			if t, err := time.Parse("2006-01-02", start); err == nil {
				filter["start_date"] = t
			} else {
				handler.logger.Error("Invalid start_date format", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid start_date format"}}
				json.NewEncoder(w).Encode(response)
				return
			}
		}
		if end := query.Get("end_date"); end != "" {
			if t, err := time.Parse("2006-01-02", end); err == nil {
				filter["end_date"] = t
			} else {
				handler.logger.Error("Invalid end_date format", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid end_date format"}}
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		// Pagination
		page := 1
		pageSize := 50
		if val := query.Get("page"); val != "" {
			if p, err := strconv.Atoi(val); err == nil && p > 0 {
				page = p
			} else {
				handler.logger.Error("Invalid page parameter", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid page parameter"},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
		}
		if val := query.Get("pageSize"); val != "" {
			if s, err := strconv.Atoi(val); err == nil && s > 0 && s <= 100 {
				pageSize = s
			} else {
				handler.logger.Error("Invalid pageSize parameter", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid pageSize parameter"},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		// --- Query Transactions ---
		transactions, total, err := handler.store.GetTransactions(filter, page, pageSize)
		if err != nil {
			handler.logger.Error("Error fetching transactions", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error fetching transactions"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// --- Count total for pagination ---
		totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

		// --- Response ---
		result := map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
			"data":        transactions,
		}

		handler.logger.Info("Transactions fetched successfully", zap.Int("total", total), zap.Int("page", page), zap.Int("pageSize", pageSize))
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    result,
		}
		json.NewEncoder(w).Encode(response)
	} else if orderID != "" {
		query := r.URL.Query()
		product := query.Get("product")

		if product == "" {
			handler.logger.Error("Missing product for transaction lookup")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Missing product type"},
			})
			return
		}

		transaction, err := handler.fetchTransactionByProduct(orderID, product)
		if err != nil {
			if err.Error() == "invalid product type" {
				handler.logger.Warn("Invalid product type", zap.String("product", product))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid product type"},
				})
				return
			}

			handler.logger.Error("Error fetching transaction", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error fetching transaction by order ID"},
			})
			return
		}

		handler.logger.Info("Transaction fetched successfully", zap.String("orderID", orderID))
		json.NewEncoder(w).Encode(responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": transaction},
		})
		return
	} else {
		handler.logger.Error("Invalid request parameters")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "Invalid request parameters"},
		})
	}
}

func (handler *HttpHandler) fetchTransactionByProduct(orderID, product string) (interface{}, error) {
	store := handler.store

	switch product {
	case "airtime":
		return store.GetAirtimeTransactionDetails(orderID)

	case "data":
		// Try Spectranet, Smile, or default Data in order
		if res, err := store.GetSpecTransDetails(orderID); err == nil {
			return res, nil
		}
		if res, err := store.GetSmileTransDetails(orderID); err == nil {
			return res, nil
		}
		return store.GetDataTransactionDetails(orderID)

	case "deposit-transaction":
		return store.GetDepositDetails(orderID)

	case "transfer":
		return store.GetTransferDetails(orderID)

	case "edu":
		return store.GetEduTransactionDetails(orderID)

	case "tv-sub":
		return store.GetTvSubscriptionDetails(orderID)

	case "electric-sub":
		return store.GetElectricSubDetails(orderID)

	default:
		return nil, fmt.Errorf("invalid product type")
	}
}
