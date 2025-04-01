package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	// "encoding/json"
	// "net/http"

	// containertypes "github.com/docker/docker/api/types/container"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-github/v63/github"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

type itemDelegate struct{}

func (i item) FilterValue() string { return "" }

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	List     list.Model
	Choice   string
	Quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.List.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.List.SelectedItem().(item)
			if ok {
				m.Choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.Choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.Choice))
	}
	if m.Quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + m.List.View()
}

type Image struct {
	Id        int
	CommitSHA string
	ImageID   string
}
type Repositories struct {
	Id            int64  `json:"id"`
	Owner         string `json:"owner"`
	RepoName      string `json:"repoName"`
	FullName      string `json:"fullName"`
	Commit        string `json:"commit"`
	PrDescription string `json:"prDescription"`
}

func connectToMySQL() (*sql.DB, error) {
	var Reset = "\033[0m"
	var Green = "\033[32m"
	// var user string = "MYSQL_USER"
	// var password string = "MYSQL_ROOT_PASSWORD"
	// var database string = "MYSQL_DATABASE"
	// host := "MYSQL_HOST"

	// Create the DSN (Data Source Name)
	// dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", user, password, host, database)

	// Open the connection to the MySQL database
	// db, err := sql.Open("mysql", dsn)
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Create the table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS images (
		id INT AUTO_INCREMENT PRIMARY KEY,
		commit_SHA VARCHAR(255),
		PR_Description VARCHAR(255),
		image_id VARCHAR(255),
		image_size INT,
		image_tag VARCHAR(255),
		timestamp VARCHAR(255)
		`)
	if err != nil {
		log.Fatalf("Error creating table: %v\n", err)
	}

	fmt.Println("------------------------------------------------------------------------------------------------")
	println(Green + "Connected to MySQL database" + Reset)
	return db, nil
}

func imageHandler() string {
	var image string = "imageID"
	return image
}

func timeStampHandler() string {
	var timeStamp string = "00:00"
	return timeStamp
}

func imageSizeHandler() int {
	var imageSize int = 0000
	return imageSize
}

func imageTagHandler() string {
	var imageTag string = "image Tag"
	return imageTag
}

// Write a fuction that logs into Github
func loginToGithub() {
	// Add styling to logging
	var (
		Green = "\033[32m"
		Reset = "\033[0m"
		// Yellow = "\033[33m"
	)

	// fmt.Println("------------------------------------------------------------------------------------------------")
	// println(Yellow + "Logging into Github..." + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")

	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	repoData, _, err := client.Repositories.Get(context.Background(), owner, repo)
	if err != nil {
		log.Fatal(err)
	}
	branch := "main"
	commit, _, err := client.Repositories.GetCommit(context.Background(), owner, repo, branch, nil)
	if err != nil {
		log.Fatal(err)
	}

	println(Green + "Logged into Github" + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")
	// fmt.Printf("Last commit on main branch:\n")
	fmt.Printf("Last Full commit message on main branch: %s\n", commit.GetCommit().GetMessage())
	// fmt.Printf("Author: %s\n", commit.GetCommit().GetAuthor().GetName())
	fmt.Printf("Date: %s\n", commit.GetCommit().GetAuthor().GetDate())
	// fmt.Printf("ID: %d\n", repoData.GetID())
	// fmt.Printf("repo: %+v\n", repoData.GetFullName())
	fmt.Printf("Owner: %v\n", repoData.GetOwner())
	fmt.Printf("UpdatedAt: %v\n", repoData.GetUpdatedAt())
	// fmt.Printf("SHA: %s\n", commit.GetSHA())
	fmt.Printf("PushedAt: %v\n", repoData.GetPushedAt())
	// Create a code break
	fmt.Println("------------------------------------------------------------------------------------------------")
	// fmt.Printf("Size: %d\n", repoData.GetSize())
	// fmt.Printf("CommitsURL: %s\n", repoData.GetCommitsURL())
	// fmt.Printf("FullName: %s\n", repoData.GetFullName())
	// fmt.Printf("Name: %s\n", repoData.GetName())
	// fmt.Printf("Description: %s\n", repoData.GetDescription())
	// fmt.Printf("BranchesURL: %s\n", repoData.GetBranchesURL())
	// fmt.Printf("CreatedAt: %v\n", repoData.GetCreatedAt())
	// fmt.Printf("URL: %s\n", repoData.GetURL())
	// fmt.Println("Logged into Github")
}

func characterStripper(str string) string {
	if len(str) > 35 {
		return str[:35]
	} else if (len(str)) < 34 {
		for i := len(str); i < 34; i++ {
			str += " "
		}
		return str + " "
	} else {
		return str
	}
}

func main() {
	items := []list.Item{
		item("Create an image"),
		item("Delete an image"),
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Docker image management:"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	var (
		Reset   = "\033[0m"
		Magenta = "\033[35m"
	)

	image := imageHandler()
	imageSize := imageSizeHandler()
	imageTag := imageTagHandler()

	loginToGithub()

	// CONNECT TO MYSQL
	db, err := connectToMySQL()
	if err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}

	// fmt.Println(Magenta + "-----------------------------------------------------------------------------------------------" + Reset)
	// fmt.Println(Magenta + "                           Local Container Registry          " + Reset)
	// fmt.Println(Magenta + "-----------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------" + Reset)
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
	fmt.Println(Magenta + "             \\::/    /        \\::/    /                    \\:|   |          " + Reset)
	fmt.Println(Magenta + "              \\/____/ocal      \\/____/ontainer              \\|___|egistry          " + Reset)
	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta+" |", "                Commit SHA                 |            ", "PR Description            |", "  Image ID   | ", "  Image Size   | ", "  Image Tag   |"+Reset)
	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------------------------------------------------------------------------------------" + Reset)

	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	branch := os.Getenv("GITHUB_BRANCH")

	commit, _, err := client.Repositories.GetCommit(context.Background(), owner, repo, branch, nil)
	if err != nil {
		log.Fatalf("Error getting commit: %v\n", err)
	}
	pr_description := commit.Commit.GetMessage()

	// CREATE A NEW RECORD
	_, err = db.Exec("INSERT INTO images.images (commit_SHA, PR_Description) VALUES(?, ?);", commit.GetSHA(), pr_description)
	if err != nil {
		log.Fatalf("Error inserting record: %v\n", err)
	}

	// READ FROM THE DATABASE
	// Execute a SELECT statement
	rowss, errr := db.Query("SELECT commit_SHA, PR_Description FROM images")
	if errr != nil {
		log.Fatalf("Error executing query: %v\n", errr)
	}
	defer rowss.Close()

	// Process the results
	for rowss.Next() {
		var commitSHA sql.NullString
		var description sql.NullString
		var trimmed string

		err := rowss.Scan(&commitSHA, &description)
		if err != nil {
			log.Fatalf("Error scanning row: %v\n", err)
		}

		if description.Valid {
			trimmed = characterStripper(description.String)
			// fmt.Println("Description:", trimmed)
		}

		fmt.Println(" | ", commitSHA.String, " | ", trimmed, " | ", image, " | ", imageSize, " | ", imageTag, " | ")
	}

	// Check for errors after looping through rows
	if err := rowss.Err(); err != nil {
		log.Fatalf("Error with rowss: %v\n", err)
	}

	defer db.Close()

	columns := []table.Column{
		{Title: "Commit-SHA", Width: 20},
		{Title: "PR-Description", Width: 15},
		{Title: "Image-ID", Width: 15},
		{Title: "Image Size", Width: 10},
		{Title: "Image Tag", Width: 15},
	}
	// TODO: List the following:
	// TODO: [Tabs] - [Github] Commits List the Github Commit SHA
	// TODO: [Tabs] - [Github] Commits List the Github PR-Description

	// TODO: [Tabs] - [Docker] List The Docker Image IDs
	// TODO: [Tabs] - [Docker] List The Docker Image Size
	// TODO: [Tabs] - [Docker] List The Docker Image Tags(If available)
	// TODO: [Tabs] - [Docker] Delete The Docker Image
	// TODO: [Tabs] - [Docker] Delete The Docker Container
	// TABS: [Tabs] - [Deployment] - Pull
	// TABS: [Tabs] - [Deployment] - List
	// TABS: [Tabs] - [Deployment] - Push
	// TABS: [Tabs] - [Deployment] - Delete

	// ADD Dynamic tabs to each section based on what is selected(add, edit, delete, deploy).

	rows := []table.Row{
		{"1", "Tokyo", "Japan", "37,274,000", "testdata"},
		{"2", "Delhi", "India", "32,065,760", "testdata"},
		{"3", "Shanghai", "China", "28,516,904", "testdata"},
		{"4", "Dhaka", "Bangladesh", "22,478,116", "testdata"},
		{"5", "São Paulo", "Brazil", "22,429,800", "testdata"},
		{"6", "Mexico City", "Mexico", "22,085,140", "testdata"},
		{"7", "Cairo", "Egypt", "21,750,020", "testdata"},
		{"8", "Beijing", "China", "21,333,332", "testdata"},
		{"9", "Mumbai", "India", "20,961,472", "testdata"},
		{"10", "Osaka", "Japan", "19,059,856", "testdata"},
		{"11", "Chongqing", "China", "16,874,740", "testdata"},
		{"12", "Karachi", "Pakistan", "16,839,950", "testdata"},
		{"13", "Istanbul", "Turkey", "15,636,243", "testdata"},
		{"14", "Kinshasa", "DR Congo", "15,628,085", "testdata"},
		{"15", "Lagos", "Nigeria", "15,387,639", "testdata"},
		{"16", "Buenos Aires", "Argentina", "15,369,919", "testdata"},
		{"17", "Kolkata", "India", "15,133,888", "testdata"},
		{"18", "Manila", "Philippines", "14,406,059", "testdata"},
		{"19", "Tianjin", "China", "14,011,828", "testdata"},
		{"20", "Guangzhou", "China", "13,964,637", "testdata"},
		{"21", "Rio De Janeiro", "Brazil", "13,634,274", "testdata"},
		{"22", "Lahore", "Pakistan", "13,541,764", "testdata"},
		{"23", "Bangalore", "India", "13,193,035", "testdata"},
		{"24", "Shenzhen", "China", "12,831,330", "testdata"},
		{"25", "Moscow", "Russia", "12,640,818", "testdata"},
		{"26", "Chennai", "India", "11,503,293", "testdata"},
		{"27", "Bogota", "Colombia", "11,344,312", "testdata"},
		{"28", "Paris", "France", "11,142,303", "testdata"},
		{"29", "Jakarta", "Indonesia", "11,074,811", "testdata"},
		{"30", "Lima", "Peru", "11,044,607", "testdata"},
		{"31", "Bangkok", "Thailand", "10,899,698", "testdata"},
		{"32", "Hyderabad", "India", "10,534,418", "testdata"},
		{"33", "Seoul", "South Korea", "9,975,709", "testdata"},
		{"34", "Nagoya", "Japan", "9,571,596", "testdata"},
		{"35", "London", "United Kingdom", "9,540,576", "testdata"},
		{"36", "Chengdu", "China", "9,478,521", "testdata"},
		{"37", "Nanjing", "China", "9,429,381", "testdata"},
		{"38", "Tehran", "Iran", "9,381,546", "testdata"},
		{"39", "Ho Chi Minh City", "Vietnam", "9,077,158", "testdata"},
		{"40", "Luanda", "Angola", "8,952,496", "testdata"},
		{"41", "Wuhan", "China", "8,591,611", "testdata"},
		{"42", "Xi An Shaanxi", "China", "8,537,646", "testdata"},
		{"43", "Ahmedabad", "India", "8,450,228", "testdata"},
		{"44", "Kuala Lumpur", "Malaysia", "8,419,566", "testdata"},
		{"45", "New York City", "United States", "8,177,020", "testdata"},
		{"46", "Hangzhou", "China", "8,044,878", "testdata"},
		{"47", "Surat", "India", "7,784,276", "testdata"},
		{"48", "Suzhou", "China", "7,764,499", "testdata"},
		{"49", "Hong Kong", "Hong Kong", "7,643,256", "testdata"},
		{"50", "Riyadh", "Saudi Arabia", "7,538,200", "testdata"},
		{"51", "Shenyang", "China", "7,527,975", "testdata"},
		{"52", "Baghdad", "Iraq", "7,511,920", "testdata"},
		{"53", "Dongguan", "China", "7,511,851", "testdata"},
		{"54", "Foshan", "China", "7,497,263", "testdata"},
		{"55", "Dar Es Salaam", "Tanzania", "7,404,689", "testdata"},
		{"56", "Pune", "India", "6,987,077", "testdata"},
		{"57", "Santiago", "Chile", "6,856,939", "testdata"},
		{"58", "Madrid", "Spain", "6,713,557", "testdata"},
		{"59", "Haerbin", "China", "6,665,951", "testdata"},
		{"60", "Toronto", "Canada", "6,312,974", "testdata"},
		{"61", "Belo Horizonte", "Brazil", "6,194,292", "testdata"},
		{"62", "Khartoum", "Sudan", "6,160,327", "testdata"},
		{"63", "Johannesburg", "South Africa", "6,065,354", "testdata"},
		{"64", "Singapore", "Singapore", "6,039,577", "testdata"},
		{"65", "Dalian", "China", "5,930,140", "testdata"},
		{"66", "Qingdao", "China", "5,865,232", "testdata"},
		{"67", "Zhengzhou", "China", "5,690,312", "testdata"},
		{"68", "Ji Nan Shandong", "China", "5,663,015", "testdata"},
		{"69", "Barcelona", "Spain", "5,658,472", "testdata"},
		{"70", "Saint Petersburg", "Russia", "5,535,556", "testdata"},
		{"71", "Abidjan", "Ivory Coast", "5,515,790", "testdata"},
		{"72", "Yangon", "Myanmar", "5,514,454", "testdata"},
		{"73", "Fukuoka", "Japan", "5,502,591", "testdata"},
		{"74", "Alexandria", "Egypt", "5,483,605", "testdata"},
		{"75", "Guadalajara", "Mexico", "5,339,583", "testdata"},
		{"76", "Ankara", "Turkey", "5,309,690", "testdata"},
		{"77", "Chittagong", "Bangladesh", "5,252,842", "testdata"},
		{"78", "Addis Ababa", "Ethiopia", "5,227,794", "testdata"},
		{"79", "Melbourne", "Australia", "5,150,766", "testdata"},
		{"80", "Nairobi", "Kenya", "5,118,844", "testdata"},
		{"81", "Hanoi", "Vietnam", "5,067,352", "testdata"},
		{"82", "Sydney", "Australia", "5,056,571", "testdata"},
		{"83", "Monterrey", "Mexico", "5,036,535", "testdata"},
		{"84", "Changsha", "China", "4,809,887", "testdata"},
		{"85", "Brasilia", "Brazil", "4,803,877", "testdata"},
		{"86", "Cape Town", "South Africa", "4,800,954", "testdata"},
		{"87", "Jiddah", "Saudi Arabia", "4,780,740", "testdata"},
		{"88", "Urumqi", "China", "4,710,203", "testdata"},
		{"89", "Kunming", "China", "4,657,381", "testdata"},
		{"90", "Changchun", "China", "4,616,002", "testdata"},
		{"91", "Hefei", "China", "4,496,456", "testdata"},
		{"92", "Shantou", "China", "4,490,411", "testdata"},
		{"93", "Xinbei", "Taiwan", "4,470,672", "testdata"},
		{"94", "Kabul", "Afghanistan", "4,457,882", "testdata"},
		{"95", "Ningbo", "China", "4,405,292", "testdata"},
		{"96", "Tel Aviv", "Israel", "4,343,584", "testdata"},
		{"97", "Yaounde", "Cameroon", "4,336,670", "testdata"},
		{"98", "Rome", "Italy", "4,297,877", "testdata"},
		{"99", "Shijiazhuang", "China", "4,285,135", "testdata"},
		{"100", "Montreal", "Canada", "4,276,526", "testdata"},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	// headerStyle := baseStyle.Foreground(lipgloss.Color("252")).Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	// m := model{t}
	// if _, err := tea.NewProgram(m).Run(); err != nil {
	// 	fmt.Println("Error running program:", err)
	// 	os.Exit(1)
	// }
}
