package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	utils "github.com/anthonygilbertt/local-container-registry/src"

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

// This init() function loads in the .env file into environment variables

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not load it:", err)
	}
}

var db *sql.DB

func main() {
	
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
	// repoData, _, err := client.Repositories.Get(context.Background(), owner, repo)
	// _, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Repository Name: %s\n", repoData.GetName())
	// fmt.Printf("Repository Description: %s\n", repoData.GetDescription())
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

	fmt.Println(utils.Magenta + " -----------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(utils.Magenta + "            _____            _____                         _____          " + Reset)
	fmt.Println(utils.Magenta + "           /\\    \\          /\\    \\                       /\\    \\         " + Reset)
	fmt.Println(utils.Magenta + "          /::\\____\\        /::\\    \\                     /::\\    \\        " + Reset)
	fmt.Println(utils.Magenta + "         /:::/    /       /::::\\    \\                   /::::\\    \\       " + Reset)
	fmt.Println(utils.Magenta + "        /:::/    /       /::::::\\    \\                 /::::::\\    \\      " + Reset)
	fmt.Println(utils.Magenta + "       /:::/    /       /:::/\\:::\\    \\               /:::/\\:::\\    \\     " + Reset)
	fmt.Println(utils.Magenta + "      /:::/    /       /:::/  \\:::\\    \\             /:::/__\\:::\\    \\    " + Reset)
	fmt.Println(utils.Magenta + "     /:::/    /       /:::/    \\:::\\    \\           /::::\\   \\:::\\    \\   " + Reset)
	fmt.Println(utils.Magenta + "    /:::/    /       /:::/    / \\:::\\    \\         /::::::\\   \\:::\\    \\  " + Reset)
	fmt.Println(utils.Magenta + "   \\:::/    /        /:::/    /   \\:::\\    \\      /:::/\\:::\\   \\:::\\____\\ " + Reset)
	fmt.Println(utils.Magenta + "    \\:::/__/         /:::/____/     \\:::\\____\\    /:::/  \\:::\\   \\:::|    |" + Reset)
	fmt.Println(utils.Magenta + "     \\:::\\   \\       \\:::\\    \\      \\  /     /  /:::/   |::::\\  /:::|____|" + Reset)
	fmt.Println(utils.Magenta + "      \\:::\\   \\       \\:::\\    \\      \\/_____/  /___/    |:::::\\/:::/    / " + Reset)
	fmt.Println(utils.Magenta + "       \\:::\\   \\       \\:::\\    \\                        |:::::::::/    /  " + Reset)
	fmt.Println(utils.Magenta + "        \\:::\\   \\       \\:::\\    \\                       |::|\\::::/    /   " + Reset)
	fmt.Println(utils.Magenta + "         \\:::\\   \\       \\:::\\    \\                      |::| \\::/____/    " + Reset)
	fmt.Println(utils.Magenta + "          \\:::\\   \\       \\:::\\    \\                     |::|  ~|          " + Reset)
	fmt.Println(utils.Magenta + "           \\:::\\   \\       \\:::\\    \\                    |::|   |          " + Reset)
	fmt.Println(utils.Magenta + "            \\:::\\___\\       \\:::\\____\\                   \\::|   |          " + Reset)
	fmt.Println(utils.Magenta + "             \\::/    /        \\::/    /                   \\:|   |          " + Reset)
	fmt.Println(utils.Magenta + "              \\/____/ocal      \\/____/ontainer             \\|___|egistry          " + Reset)
	fmt.Println(utils.Magenta + " -----------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(utils.Magenta+" |", "                Commit SHA                 |                   ", "PR Description                   |", "  Image ID   | ", "  Image Size   | ", "  Image Tag   |"+Reset)
	fmt.Println(utils.Magenta + " |----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|" + Reset)
	// Display commits in the ASCII table
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()
		fmt.Println(utils.Magenta+" |  ", commit.GetSHA(), "|", commitMessage, "|-----------------|--------------------|-------------------|-----------------------|"+Reset)
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

	// Start TUI with collected data from all commits
	var tableData []TableData
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()
		tableData = append(tableData, TableData{
			CommitSHA:     commit.GetSHA(),
			PRDescription: commitMessage,
			ImageID:       "N/A",
			ImageSize:     "N/A", 
			ImageTag:      "N/A",
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
