//package smsclient
//
//import (
//	"encoding/json"
//	"fmt"
//	"math/rand"
//	"net/http"
//	"time"
//
//	"github.com/go-chi/chi"
//	"github.com/twilio/twilio-go"
//	"github.com/twilio/twilio-go/rest/api/v2010/account/message"
//	"github.com/twinj/uuid"
//)
//
//var (
//	twilioAccountSid = "your_twilio_account_sid"
//	twilioAuthToken  = "your_twilio_auth_token"
//)
//
//var otpDataStore = make(map[string]OTPData)
//
//func generateOTP() string {
//	rand.Seed(time.Now().UnixNano())
//	return fmt.Sprintf("%06d", rand.Intn(1000000))
//}
//
//func createOTPHandler(w http.ResponseWriter, r *http.Request) {
//	phoneNumber := r.FormValue("phone_number")
//
//	otpID := uuid.NewV4().String()
//	otp := generateOTP()
//
//	otpDataStore[otpID] = OTPData{
//		ID:       otpID,
//		OTP:      otp,
//		Verified: false,
//	}
//
//	client := twilio.NewRestClient(twilioAccountSid, twilioAuthToken)
//	messageService := client.ApiV2010.CreateMessage(message.CreateMessageParams{
//		To:   twilio.String(phoneNumber),
//		From: twilio.String("your_twilio_phone_number"),
//		Body: twilio.String(fmt.Sprintf("Your OTP is: %s", otp)),
//	})
//
//	if messageService != nil {
//		w.WriteHeader(http.StatusCreated)
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"message": "OTP sent successfully"})
//	} else {
//		w.WriteHeader(http.StatusInternalServerError)
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to send OTP"})
//	}
//}
//
//func verifyOTPHandler(w http.ResponseWriter, r *http.Request) {
//	otpID := chi.URLParam(r, "otpID")
//	otp := r.FormValue("otp")
//
//	storedOTPData, found := otpDataStore[otpID]
//	if !found {
//		w.WriteHeader(http.StatusNotFound)
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"error": "OTP not found"})
//		return
//	}
//
//	if storedOTPData.OTP == otp && !storedOTPData.Verified {
//		storedOTPData.Verified = true
//		otpDataStore[otpID] = storedOTPData
//
//		w.WriteHeader(http.StatusOK)
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"message": "OTP verified successfully"})
//	} else {
//		w.WriteHeader(http.StatusUnauthorized)
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid OTP"})
//	}
//}
