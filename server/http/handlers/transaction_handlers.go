package handlers

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		// Filters
		filter := make(map[string]interface{})
		filter["user_id"] = id

		// Basic filters
		if flow := query.Get("flow"); flow != "" {
			filter["flow"] = flow
		}
		if category := query.Get("category"); category != "" {
			filter["category"] = category
		}
		if subcategory := query.Get("subcategory"); subcategory != "" {
			filter["subcategory"] = subcategory
		}

		if status := query.Get("status"); status != "" {
			filter["status"] = status
		}

		datefilter, err := addDateFilters(query)
		if err != nil {
			handler.logger.Error("Invalid date format", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		for k, v := range datefilter {
			filter[k] = v
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
		transactions, err := handler.store.GetTransactions(filter, page, pageSize)
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
		totalPages := int(math.Ceil(float64(transactions.TotalCount) / float64(pageSize)))

		// --- Response ---
		result := map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       transactions.TotalCount,
			"total_pages": totalPages,
			"data":        transactions,
		}

		handler.logger.Info("Transactions fetched successfully", zap.Int("total", transactions.TotalCount), zap.Int("page", page), zap.Int("pageSize", pageSize))
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

func (handler *HttpHandler) GetWalletSummary(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	query := r.URL.Query()
	page := 1
	pageSize := 50

	filter := make(map[string]interface{})

	datefilter, err := addDateFilters(query)
	if err != nil {
		handler.logger.Error("Invalid date format", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	for k, v := range datefilter {
		filter[k] = v
	}

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

	if record := query.Get("record"); record != "" {
		filter["record"] = record
	}

	if status := query.Get("status"); status != "" {
		filter["status"] = status
	}

	if category := query.Get("category"); category != "" {
		filter["category"] = category
	}

	filter["user_id"] = id

	summary, err := handler.store.GetWalletSummary(filter, page)
	if err != nil {
		handler.logger.Error("Error fetching wallet summary", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error fetching wallet summary"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	totalPages := int(math.Ceil(float64(summary.TotalCount) / float64(pageSize)))

	result := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total":       summary.TotalCount,
		"total_pages": totalPages,
		"data":        summary,
	}

	handler.logger.Info("Wallet summary fetched successfully", zap.Int("total", summary.TotalCount), zap.Int("page", page), zap.Int("pageSize", pageSize))
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": result},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) GetSalesSummary(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	query := r.URL.Query()
	category := query.Get("category")
	page := 1

	filter := map[string]interface{}{
		"user_id": id,
	}

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

	datefilter, err := addDateFilters(query)
	if err != nil {
		handler.logger.Error("Invalid date format", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	for k, v := range datefilter {
		filter[k] = v
	}

	summary, err := handler.store.GetSalesSummary(category, filter, page)
	if err != nil {
		handler.logger.Error("Error fetching sales summary", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error fetching sales summary"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	totalPages := int(math.Ceil(float64(summary.TotalCount) / float64(50)))

	result := map[string]interface{}{
		"page":        page,
		"page_size":   50,
		"total":       summary.TotalCount,
		"total_pages": totalPages,
		"data":        summary,
	}

	handler.logger.Info("Sales summary fetched successfully", zap.Int("total", summary.TotalCount), zap.Int("page", page))
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": result},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetSalesOverview(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	filter := map[string]interface{}{
		"user_id": userDetails.ID,
	}

	query := r.URL.Query()
	datefilter, err := addDateFilters(query)
	if err != nil {
		handler.logger.Error("Invalid date format", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	for k, v := range datefilter {
		filter[k] = v
	}

	overview, err := handler.store.GetSalesOverview(filter)
	if err != nil {
		handler.logger.Error("Error fetching sales overview", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error fetching sales overview"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	handler.logger.Info("Sales overview fetched successfully", zap.Int("total_product", overview.TotalProduct))
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": overview},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func addDateFilters(q url.Values) (filter map[string]interface{}, err error) {
	filter = make(map[string]interface{})
	if start := q.Get("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			filter["start_date"] = t
		} else {
			return nil, fmt.Errorf("invalid start_date format")
		}
	}

	if end := q.Get("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			filter["end_date"] = t
		} else {
			return nil, fmt.Errorf("invalid end_date format")
		}
	}
	return filter, nil
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

	case "deposit":
		return store.GetDepositDetails(orderID)

	case "transfer":
		return store.GetTransferDetails(orderID)

	case "edu":
		return store.GetEduTransactionDetails(orderID)

	case "tv-sub":
		return store.GetTvSubscriptionDetails(orderID)

	case "electric-sub":
		return store.GetElectricSubDetails(orderID)

	case "point":
		return store.GetPointRedeemDetails(orderID)

	default:
		return nil, fmt.Errorf("invalid product type")
	}
}

func (handler *HttpHandler) Chart(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	query := r.URL.Query()

	rangeType := strings.ToUpper(strings.TrimSpace(query.Get("range")))
	filter := make(map[string]interface{})

	datefilter, err := addDateFilters(query)
	if err != nil {
		handler.logger.Error("Invalid date format", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	maps.Copy(filter, datefilter)

	filter["user_id"] = userDetails.ID

	if rangeType == "" && len(datefilter) == 0 {
		rangeType = "DAILY"
	}

	chartData, err := handler.store.GetChart(filter, rangeType)
	if err != nil {
		handler.logger.Error("Failed to get chart data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": chartData},
	}
	handler.logger.Info("Chart data retrieved successfully", zap.String("user_id", userDetails.ID), zap.String("range_type", rangeType))

	json.NewEncoder(w).Encode(response)
}
