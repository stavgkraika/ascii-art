package main

import (
	"ascii-art/internal/ascii"
	"encoding/json"
	"log"
	"net/http"
)

// ============================================================================
// REST API DATA STRUCTURES
// ============================================================================

// Request represents the JSON request body for ASCII art generation
// REST API: Accepts POST requests with JSON body containing text and optional parameters
type Request struct {
	Text      string `json:"text"`      // Required: text to convert to ASCII art
	Banner    string `json:"banner"`    // Optional: banner style (standard, shadow, thinkertoy)
	Color     string `json:"color,omitempty"`     // Optional: color name (red, green, blue, etc.)
	Substring string `json:"substring,omitempty"` // Optional: substring to colorize
	Align     string `json:"align,omitempty"`     // Optional: alignment (left, right, center, justify)
}

// Response represents the successful JSON response
// REST API: Returns JSON with ASCII art result on success (200 OK)
type Response struct {
	Result string `json:"result"` // Generated ASCII art as string
}

// ErrorResponse represents the error JSON response
// REST API: Returns JSON with error message on failure (400, 404, 500)
type ErrorResponse struct {
	Error string `json:"error"` // Error description
}

// setupHandler configures HTTP routes
func setupHandler() {
	http.HandleFunc("/ascii-art", asciiArtHandler)
	http.HandleFunc("/server.html", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Serving server.html from: docs/server.html")
		http.ServeFile(w, r, "docs/server.html")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 - Use /server.html or /ascii-art"))
	})
}

func main() {
	setupHandler()
	
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// ============================================================================
// REST API ENDPOINT: POST /ascii-art
// ============================================================================
// asciiArtHandler handles POST requests to /ascii-art endpoint
// This is the main REST API endpoint for ASCII art generation
//
// REST API Specification:
// - Endpoint: POST /ascii-art
// - Content-Type: application/json
// - Request Body: JSON with text (required), banner, color, substring, align (optional)
// - Response: JSON with result or error
//
// Returns appropriate HTTP status codes:
// - 200 OK: Successful ASCII art generation
// - 400 Bad Request: Invalid input (wrong method, invalid JSON, missing text, invalid alignment)
// - 404 Not Found: Banner file not found
// - 500 Internal Server Error: Generation failure or JSON encoding error
func asciiArtHandler(w http.ResponseWriter, r *http.Request) {
	// REST API: Set CORS headers to allow cross-origin requests from web clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// REST API: Handle preflight OPTIONS request for CORS
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// REST API STATUS 400: Invalid HTTP method (only POST allowed)
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusBadRequest)
		return
	}

	// REST API: Parse JSON request body into Request struct
	// STATUS 400: Invalid JSON in request body
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// REST API: Validate required field
	// STATUS 400: Missing required text field
	if req.Text == "" {
		sendError(w, "Text is required", http.StatusBadRequest)
		return
	}

	// REST API: Set default banner if not specified
	if req.Banner == "" {
		req.Banner = "standard"
	}

	// REST API: Validate optional alignment parameter
	// STATUS 400: Invalid alignment value
	if req.Align != "" && !isValidAlignment(req.Align) {
		sendError(w, "Invalid alignment", http.StatusBadRequest)
		return
	}

	// REST API: Load banner template
	// STATUS 404: Banner file not found
	bannerFile := "assets/" + req.Banner + ".txt"
	charMap, err := ascii.LoadBanner(bannerFile)
	if err != nil {
		sendError(w, "Banner not found", http.StatusNotFound)
		return
	}

	// REST API: Generate ASCII art with requested parameters
	result := ascii.GenerateArtWithColorAndAlignment(req.Text, charMap, req.Substring, req.Color, req.Align)
	
	// REST API STATUS 500: ASCII art generation failed (empty result)
	if result == "" {
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Strip ANSI color codes for HTML display
	result = stripANSI(result)
	
	// REST API STATUS 200: Success - return ASCII art result as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Response{Result: result}); err != nil {
		// REST API STATUS 500: JSON encoding failed
		sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// stripANSI removes ANSI color codes from string for HTML display
func stripANSI(s string) string {
	result := ""
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++
		} else if inEscape && s[i] == 'm' {
			inEscape = false
		} else if !inEscape {
			result += string(s[i])
		}
	}
	return result
}

// isValidAlignment checks if the alignment value is valid
func isValidAlignment(align string) bool {
	validAlignments := []string{"left", "right", "center", "justify"}
	for _, valid := range validAlignments {
		if align == valid {
			return true
		}
	}
	return false
}

// sendError sends an error response with the specified HTTP status code
// REST API: Helper function to send JSON error responses
func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
