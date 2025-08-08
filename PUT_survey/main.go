package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Models ---
type SurveyForm struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Questions   []Question         `bson:"questions" json:"questions"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
}

type Question struct {
	ID       string   `bson:"id" json:"id"`
	Text     string   `bson:"text" json:"text"`
	Type     string   `bson:"type" json:"type"`
	Options  []string `bson:"options,omitempty" json:"options,omitempty"`
	Required bool     `bson:"required" json:"required"`
}

// --- Handler Struct ---
type SurveyHandler struct {
	collection *mongo.Collection
}

func NewSurveyHandler(db *mongo.Database) *SurveyHandler {
	return &SurveyHandler{
		collection: db.Collection("survey_forms"),
	}
}

// --- PUT: Update Survey Form ---
func (h *SurveyHandler) UpdateSurveyForm(w http.ResponseWriter, r *http.Request) {
	var form SurveyForm

	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if form.ID.IsZero() {
		http.Error(w, "Survey ID is required", http.StatusBadRequest)
		return
	}

	userRole := r.Header.Get("X-User-Role")
	if userRole != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	update := bson.M{
		"$set": bson.M{
			"title":       form.Title,
			"description": form.Description,
			"questions":   form.Questions,
			"is_active":   form.IsActive,
		},
	}

	res, err := h.collection.UpdateByID(context.Background(), form.ID, update)
	if err != nil {
		http.Error(w, "Failed to update survey", http.StatusInternalServerError)
		return
	}

	if res.MatchedCount == 0 {
		http.Error(w, "Survey not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      form.ID.Hex(),
		"message": "Survey updated successfully",
	})
}

// --- Mongo Connection ---
func connectMongoDB() *mongo.Client {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI is not set")
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect failed: %v", err)
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	log.Println("✅ Connected to MongoDB")
	return client
}

// --- Load .env ---
func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("⚠️  .env file not found, using system environment")
	}
}

// --- Main ---
func main() {
	loadEnv()

	client := connectMongoDB()
	db := client.Database(os.Getenv("DATABASE_NAME"))
	handler := NewSurveyHandler(db)

	http.HandleFunc("/update-survey", func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Role, X-User-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != "PUT" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handler.UpdateSurveyForm(w, r)
	})

	log.Println("🚀 Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
