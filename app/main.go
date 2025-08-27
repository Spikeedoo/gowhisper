package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/go-chi/chi/v5"
)

// Global declaration
var JobMap sync.Map
var JobQueue = make(chan Job, 100)

// Types
type Job struct {
	Id         string `json:"id"`
	AudioUrl   string `json:"audioUrl"`
	Transcript string `json:"transcript"`
}

type JobRequest struct {
	AudioUrl string `json:"audioUrl"`
}

type JobResponse struct {
	Id string `json:"id"`
}

// Functions
/* Generate random ID */
func generateId(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomString := make([]byte, length)
	for i := 0; i < length; i++ {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return string(randomString)
}

/* POST: Create transcription job */
func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Declare new job
	newJobId := generateId(12)
	newJob := Job{
		Id:         newJobId,
		AudioUrl:   req.AudioUrl,
		Transcript: "",
	}

	// Set up response payload
	newJobRes := JobResponse{
		Id: newJobId,
	}

	// Turn response payload into JSON
	jobResJson, err := json.Marshal(newJobRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store job in map
	JobMap.Store(newJobId, newJob)

	// Push to job queue
	JobQueue <- newJob

	// Set response headers & return
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jobResJson)
}

/* GET: Check transcription job */
func CheckJobHandler(w http.ResponseWriter, r *http.Request) {
	jobId := chi.URLParam(r, "jobId")

	selectedJob, ok := JobMap.Load(jobId)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	jobJson, err := json.Marshal(selectedJob)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers & return
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jobJson)
}

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func ConsumeJobQueue(JobQueue <-chan Job) {
	for job := range JobQueue {
		tempFilePath := fmt.Sprintf("/tmp/%s", job.Id)
		tempFileTranscriptPath := fmt.Sprintf("/tmp/%s.txt", job.Id)

		fmt.Printf("Downloading from %s !!!TO!!! %s ...", job.AudioUrl, tempFilePath)

		err := downloadFile(tempFilePath, job.AudioUrl)
		if err != nil {
			fmt.Println("Error downloading file:", err)
			return
		}

		cmd := exec.Command("whisper-cli", "-m", "/models/ggml-base.en.bin", "-f", tempFilePath, "--output-txt", ">", "/dev/null")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		transcript, err := ioutil.ReadFile(tempFileTranscriptPath)
		if err != nil {
			fmt.Println("Error reading transcript file:", err)
			return
		}

		updatedJob := Job{
			Id:         job.Id,
			AudioUrl:   job.AudioUrl,
			Transcript: string(transcript),
		}

		JobMap.Swap(job.Id, updatedJob)
	}
}

func main() {
	// Set up Chi router
	router := chi.NewRouter()

	// Routes
	router.Post("/job", CreateJobHandler)
	router.Get("/job/{jobId}", CheckJobHandler)

	// Start job queue
	go ConsumeJobQueue(JobQueue)

	// Start server
	err := http.ListenAndServe(":3000", router)
	if err != nil {
		log.Fatal("Server failed to start")
	}

	// Close job queue
	close(JobQueue)
}
