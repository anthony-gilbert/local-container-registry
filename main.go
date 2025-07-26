package main

//go:generate go run build.go

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/google/go-github/v63/github"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Repositories struct {
	id            int64
	owner         string
	repoName      string
	fullName      string
	commit        string
	prDescription string
}

type DockerImage struct {
	ID       string
	RepoTags []string
	Size     string
}

type TableData struct {
	CommitSHA     string
	PRDescription string
	ImageID       string
	ImageSize     string
	ImageTag      string
	PushedAt      string
}

// This init() function loads in the .env file into environment variables

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not load it:", err)
	}
}

var db *sql.DB

func getDockerImageInfo() (*DockerImage, error) {
	// Use docker images with custom format to get the info we need
	cmd := exec.Command("docker", "images", "--format", "{{.ID}},{{.Repository}}:{{.Tag}},{{.Size}}", "local-container-registry")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker images: %v", err)
	}

	if len(output) == 0 {
		return &DockerImage{
			ID:       "Not Found",
			RepoTags: []string{"N/A"},
			Size:     "N/A",
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return &DockerImage{
			ID:       "Not Found",
			RepoTags: []string{"N/A"},
			Size:     "N/A",
		}, nil
	}

	// Parse the first line (most recent image)
	parts := strings.Split(lines[0], ",")
	if len(parts) >= 3 {
		return &DockerImage{
			ID:       parts[0],
			RepoTags: []string{parts[1]},
			Size:     parts[2], // Size as string from docker images
		}, nil
	}

	return &DockerImage{
		ID:       "Parse Error",
		RepoTags: []string{"N/A"},
		Size:     "N/A",
	}, nil
}

func main() {
	// Check if DOCKER_BUILD environment variable is set
	if os.Getenv("DOCKER_BUILD") == "true" {
		fmt.Println("🐳 Building Docker image...")

		cmd := exec.Command("docker", "build", "-t", "local-container-registry", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Fatalf("❌ Docker build failed: %v", err)
		}

		fmt.Println("✅ Docker image built successfully!")
		fmt.Println("🚀 You can now run: docker run --rm -it local-container-registry")
		return
	}

	// Capture connection properties for the MySQL database
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	cfg.Passwd = os.Getenv("MYSQL_ROOT_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	cfg.DBName = "images"

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	var (
		Green  = "\033[32m"
		Reset  = "\033[0m"
		Yellow = "\033[33m"
	)

	fmt.Println("------------------------------------------------------------------------------------------------")
	println(Yellow + "Logging into Github..." + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")

	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	
	// Get repository data
	repoData, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Printf("Warning: Could not get repository data: %v", err)
	}
	
	branch := "master"
	// Get multiple commits instead of just one
	commits, _, err := client.Repositories.ListCommits(context.Background(), owner, repo, &github.CommitsListOptions{
		SHA: branch,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 10, // Get last 10 commits
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	println(Green + "Logged into Github" + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")
	fmt.Printf("Found %d commits on master branch\n", len(commits))
	if len(commits) > 0 {
		fmt.Printf("Latest commit: %s\n", commits[0].GetCommit().GetMessage())
		fmt.Printf("SHA: %s\n", commits[0].GetSHA())
	}

	// fmt.Printf("Owner: %v\n", repoData.GetOwner())
	// fmt.Printf("repo: %+v\n", repoData.GetFullName())
	// fmt.Printf("Date: %s\n", commit.GetCommit().GetAuthor().GetDate())
	// fmt.Printf("UpdatedAt: %v\n", repoData.GetUpdatedAt())
	// fmt.Printf("Author: %s\n", commit.GetCommit().GetAuthor().GetName())
	// fmt.Printf("ID: %d\n", repoData.GetID())
	// 	fmt.Printf("PushedAt: %v\n", repoData.GetPushedAt())
	// 	Create a code break
	// 	fmt.Println("------------------------------------------------------------------------------------------------")
	// 	fmt.Printf("Size: %d\n", repoData.GetSize())
	// 	fmt.Printf("CommitsURL: %s\n", repoData.GetCommitsURL())
	// 	fmt.Printf("FullName: %s\n", repoData.GetFullName())
	// 	fmt.Printf("Name: %s\n", repoData.GetName())
	// 	fmt.Printf("Description: %s\n", repoData.GetDescription())
	// 	fmt.Printf("BranchesURL: %s\n", repoData.GetBranchesURL())
	// 	fmt.Printf("CreatedAt: %v\n", repoData.GetCreatedAt())
	// 	fmt.Printf("URL: %s\n", repoData.GetURL())
	// 	fmt.Println("Logged into Github")

	// Process each commit for database insertion
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()
		fmt.Printf("Processing commit: %s\n", commitMessage)

		// Insert into MySQL database
		_, err = db.Exec("INSERT INTO images (PR_Description) VALUES (?)", commitMessage)
		if err != nil {
			log.Printf("Error inserting commit into MySQL: %v", err)
		}
	}

	// TODO: [Tabs] - [Github] List the Github Commit SHA - DONE
	// TODO: [Tabs] - [Github] List the Github PR-Description - DONE
	// TODO: [Tabs] - [Docker] List The Docker Image IDs
	// TODO: [Tabs] - [Docker] List The Docker Image Size
	// TODO: [Tabs] - [Docker] List The Docker Image Tags(If available)
	// TODO: [Tabs] - [Docker] Delete The Docker Image
	// TODO: [Tabs] - [Docker] Delete The Docker Container
	// TODO: [Tabs] - [Deployment] - Pull
	// TODO: [Tabs] - [Deployment] - List
	// TODO: [Tabs] - [Deployment] - Push
	// TODO: [Tabs] - [Deployment] - Delete

	// Get Docker image information
	dockerInfo, err := getDockerImageInfo()
	if err != nil {
		log.Printf("Warning: Could not get Docker image info: %v", err)
		dockerInfo = &DockerImage{
			ID:       "Error",
			RepoTags: []string{"N/A"},
			Size:     "N/A",
		}
	}

	// Format Docker data
	imageID := dockerInfo.ID
	if len(imageID) > 12 {
		imageID = imageID[:12] // Show short ID like Docker CLI
	}

	imageTag := "N/A"
	if len(dockerInfo.RepoTags) > 0 && dockerInfo.RepoTags[0] != "<none>:<none>" {
		imageTag = dockerInfo.RepoTags[0]
	}

	imageSize := dockerInfo.Size
	if dockerInfo.Size == "" || dockerInfo.Size == "N/A" {
		imageSize = "N/A"
	}

	// Start TUI with collected data from all commits
	var tableData []TableData
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()
		
		// Get PushedAt from repository data
		pushedAt := "N/A"
		if repoData != nil && repoData.PushedAt != nil {
			pushedAt = repoData.GetPushedAt().Format("2006-01-02 15:04:05")
		}
		
		tableData = append(tableData, TableData{
			CommitSHA:     commit.GetSHA(),
			PRDescription: commitMessage,
			ImageID:       imageID,
			ImageSize:     imageSize,
			ImageTag:      imageTag,
			PushedAt:      pushedAt,
		})
	}
	startTUI(tableData)
}

// I need to insert git commits into the mysql database
func insertIntoPostgresDB(commitSHA string, commitMessage string) {
	// Connect to the database
	db, err := sql.Open("postgres", "user=root password=new_password dbname=images sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Insert the commit SHA and PR description into the database
	fmt.Println("Inserting commit SHA and PR description into the database...")
	query := `INSERT INTO images (commit_sha, pr_description) VALUES ($1, $2)`
	_, err = db.Exec(query, commitSHA, commitMessage)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Commit SHA and PR description inserted into the database.")
	fmt.Println("Commit SHA:", commitSHA)
	fmt.Println("PR Description:", commitMessage)
}

// I need to get git commits from the MySQL database
func getFromPostgresDB() {
	// Connect to the database
	db, err := sql.Open("postgres", "user=root password=new_password dbname=images sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Get the commit SHA and PR description from the database
	query := `SELECT commit_sha, pr_description FROM images`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var commitSHA string
		var prDescription string
		err := rows.Scan(&commitSHA, &prDescription)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Commit SHA:", commitSHA)
		fmt.Println("PR Description:", prDescription)
	}
}
