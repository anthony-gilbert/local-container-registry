package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

//I want to be able to connect to a Postgres database.
// the database should have a table for github data, docker data, kubernetes/deployment data

func init() {
	// loads .env file into environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not load it:", err)
	}
}

func main() {
	var (
		Reset   = "\033[0m"
		Magenta = "\033[35m"
		Green   = "\033[32m"
		Yellow  = "\033[33m"
	)

	fmt.Println("------------------------------------------------------------------------------------------------")
	println(Yellow + "Logging into Github..." + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")

	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	// repoData, _, err := client.Repositories.Get(context.Background(), owner, repo)
	_, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Repository Name: %s\n", repoData.GetName())
	// fmt.Printf("Repository Description: %s\n", repoData.GetDescription())
	branch := "github-data-in-table"
	commit, _, err := client.Repositories.GetCommit(context.Background(), owner, repo, branch, nil)
	if err != nil {
		log.Fatal(err)
	}

	println(Green + "Logged into Github" + Reset)

	fmt.Println(Magenta + " -----------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta + "            _____            _____                         _____          " + Reset)
	fmt.Println(Magenta + "           /\\    \\         /\\    \\                       /\\    \\         " + Reset)
	fmt.Println(Magenta + "          /::\\____\\       /::\\    \\                     /::\\    \\        " + Reset)
	fmt.Println(Magenta + "         /:::/    /       /::::\\    \\                   /::::\\    \\       " + Reset)
	fmt.Println(Magenta + "        /:::/    /       /::::::\\    \\                 /::::::\\    \\      " + Reset)
	fmt.Println(Magenta + "       /:::/    /       /:::/\\:::\\    \\               /:::/\\:::\\    \\     " + Reset)
	fmt.Println(Magenta + "      /:::/    /       /:::/  \\:::\\    \\             /:::/__\\:::\\    \\    " + Reset)
	fmt.Println(Magenta + "     /:::/    /       /:::/    \\:::\\    \\           /::::\\   \\:::\\    \\   " + Reset)
	fmt.Println(Magenta + "    /:::/    /       /:::/    / \\:::\\    \\         /::::::\\   \\:::\\    \\  " + Reset)
	fmt.Println(Magenta + "   \\:::/    /        /:::/    /   \\:::\\    \\      /:::/\\:::\\   \\:::\\____\\ " + Reset)
	fmt.Println(Magenta + "    \\:::/__/         /:::/____/     \\:::\\____\\    /:::/  \\:::\\   \\:::|    |" + Reset)
	fmt.Println(Magenta + "     \\:::\\   \\       \\:::\\    \\      \\  /     /  /:::/   |::::\\  /:::|____|" + Reset)
	fmt.Println(Magenta + "      \\:::\\   \\       \\:::\\    \\      \\/_____/  /___/    |:::::\\/:::/    / " + Reset)
	fmt.Println(Magenta + "       \\:::\\   \\       \\:::\\    \\                        |:::::::::/    /  " + Reset)
	fmt.Println(Magenta + "        \\:::\\   \\       \\:::\\    \\                       |::|\\::::/    /   " + Reset)
	fmt.Println(Magenta + "         \\:::\\   \\       \\:::\\    \\                      |::| \\::/____/    " + Reset)
	fmt.Println(Magenta + "          \\:::\\   \\       \\:::\\    \\                     |::|  ~|          " + Reset)
	fmt.Println(Magenta + "           \\:::\\   \\       \\:::\\    \\                    |::|   |          " + Reset)
	fmt.Println(Magenta + "            \\:::\\___\\       \\:::\\____\\                   \\::|   |          " + Reset)
	fmt.Println(Magenta + "             \\::/    /        \\::/    /                   \\:|   |          " + Reset)
	fmt.Println(Magenta + "              \\/____/ocal      \\/____/ontainer             \\|___|egistry          " + Reset)
	fmt.Println(Magenta + " -----------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta+" |", "                Commit SHA                 |                   ", "PR Description                   |", "  Image ID   | ", "  Image Size   | ", "  Image Tag   |"+Reset)
	fmt.Println(Magenta + " |----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|" + Reset)
	fmt.Println(Magenta+" |  ", commit.GetSHA(), "|", commit.GetCommit().GetMessage(), "|-----------------|--------------------|-------------------|-----------------------|"+Reset)

	fmt.Printf("SHA: %s\n", commit.GetSHA())
	fmt.Printf("Last commit message: %s\n", commit.GetCommit().GetMessage())

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
}
