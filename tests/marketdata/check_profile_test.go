package marketdata_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
	"github.com/joho/godotenv"
)

func TestCheckProfile1(t *testing.T) {
	godotenv.Load("../../.env")
	appID := os.Getenv("FYERS_APP_ID")
	accessToken := os.Getenv("FYERS_ACCESS_TOKEN")

	client := fyersgosdk.NewFyersModel(appID, accessToken)

	profStr, err := client.GetProfile()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	var profile fyersgosdk.Profile
	if err := json.Unmarshal([]byte(profStr), &profile); err != nil {
		t.Fatalf("failed to parse profile response: %v", err)
	}

	fmt.Printf("Profile: %+v\n", profile)
}
