package main

import (
	// std lib
	"fmt"
	"io"
	"os"
	"strings"

	// external
	// _ "github.com/go-sql-griver/mysql"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).Margin(0, 0, 1, 2)
)

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

const listHeight = 14

type itemDelegate struct{}

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

func main() {

	items := []list.Item{
		item("Create an image"),
		item("Delete an image"),
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

	fmt.Println(Magenta + "------------------------------------------------------------------------------------------------" + Reset)
	fmt.Println(Magenta + "                           Local Container Registry          " + Reset)
	fmt.Println(Magenta + "-----------------------------------------------------------------------------------------------" + Reset)

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

	rows := []table.Row{
		{"1", "Tokyo", "Japan", "37,274,000", "daddy"},
		{"2", "Delhi", "India", "32,065,760", "daddy"},
		{"3", "Shanghai", "China", "28,516,904", "daddy"},
		{"4", "Dhaka", "Bangladesh", "22,478,116", "daddy"},
		{"5", "São Paulo", "Brazil", "22,429,800", "daddy"},
		{"6", "Mexico City", "Mexico", "22,085,140", "daddy"},
		{"7", "Cairo", "Egypt", "21,750,020", "daddy"},
		{"8", "Beijing", "China", "21,333,332", "daddy"},
		{"9", "Mumbai", "India", "20,961,472", "daddy"},
		{"10", "Osaka", "Japan", "19,059,856", "daddy"},
		{"11", "Chongqing", "China", "16,874,740", "daddy"},
		{"12", "Karachi", "Pakistan", "16,839,950", "daddy"},
		{"13", "Istanbul", "Turkey", "15,636,243", "daddy"},
		{"14", "Kinshasa", "DR Congo", "15,628,085", "daddy"},
		{"15", "Lagos", "Nigeria", "15,387,639", "daddy"},
		{"16", "Buenos Aires", "Argentina", "15,369,919", "daddy"},
		{"17", "Kolkata", "India", "15,133,888", "daddy"},
		{"18", "Manila", "Philippines", "14,406,059", "daddy"},
		{"19", "Tianjin", "China", "14,011,828", "daddy"},
		{"20", "Guangzhou", "China", "13,964,637", "daddy"},
		{"21", "Rio De Janeiro", "Brazil", "13,634,274", "daddy"},
		{"22", "Lahore", "Pakistan", "13,541,764", "daddy"},
		{"23", "Bangalore", "India", "13,193,035", "daddy"},
		{"24", "Shenzhen", "China", "12,831,330", "daddy"},
		{"25", "Moscow", "Russia", "12,640,818", "daddy"},
		{"26", "Chennai", "India", "11,503,293", "daddy"},
		{"27", "Bogota", "Colombia", "11,344,312", "daddy"},
		{"28", "Paris", "France", "11,142,303", "daddy"},
		{"29", "Jakarta", "Indonesia", "11,074,811", "daddy"},
		{"30", "Lima", "Peru", "11,044,607", "daddy"},
		{"31", "Bangkok", "Thailand", "10,899,698", "daddy"},
		{"32", "Hyderabad", "India", "10,534,418", "daddy"},
		{"33", "Seoul", "South Korea", "9,975,709", "daddy"},
		{"34", "Nagoya", "Japan", "9,571,596", "daddy"},
		{"35", "London", "United Kingdom", "9,540,576", "daddy"},
		{"36", "Chengdu", "China", "9,478,521", "daddy"},
		{"37", "Nanjing", "China", "9,429,381", "daddy"},
		{"38", "Tehran", "Iran", "9,381,546", "daddy"},
		{"39", "Ho Chi Minh City", "Vietnam", "9,077,158", "daddy"},
		{"40", "Luanda", "Angola", "8,952,496", "daddy"},
		{"41", "Wuhan", "China", "8,591,611", "daddy"},
		{"42", "Xi An Shaanxi", "China", "8,537,646", "daddy"},
		{"43", "Ahmedabad", "India", "8,450,228", "daddy"},
		{"44", "Kuala Lumpur", "Malaysia", "8,419,566", "daddy"},
		{"45", "New York City", "United States", "8,177,020", "daddy"},
		{"46", "Hangzhou", "China", "8,044,878", "daddy"},
		{"47", "Surat", "India", "7,784,276", "daddy"},
		{"48", "Suzhou", "China", "7,764,499", "daddy"},
		{"49", "Hong Kong", "Hong Kong", "7,643,256", "daddy"},
		{"50", "Riyadh", "Saudi Arabia", "7,538,200", "daddy"},
		{"51", "Shenyang", "China", "7,527,975", "daddy"},
		{"52", "Baghdad", "Iraq", "7,511,920", "daddy"},
		{"53", "Dongguan", "China", "7,511,851", "daddy"},
		{"54", "Foshan", "China", "7,497,263", "daddy"},
		{"55", "Dar Es Salaam", "Tanzania", "7,404,689", "daddy"},
		{"56", "Pune", "India", "6,987,077", "daddy"},
		{"57", "Santiago", "Chile", "6,856,939", "daddy"},
		{"58", "Madrid", "Spain", "6,713,557", "daddy"},
		{"59", "Haerbin", "China", "6,665,951", "daddy"},
		{"60", "Toronto", "Canada", "6,312,974", "daddy"},
		{"61", "Belo Horizonte", "Brazil", "6,194,292", "daddy"},
		{"62", "Khartoum", "Sudan", "6,160,327", "daddy"},
		{"63", "Johannesburg", "South Africa", "6,065,354", "daddy"},
		{"64", "Singapore", "Singapore", "6,039,577", "daddy"},
		{"65", "Dalian", "China", "5,930,140", "daddy"},
		{"66", "Qingdao", "China", "5,865,232", "daddy"},
		{"67", "Zhengzhou", "China", "5,690,312", "daddy"},
		{"68", "Ji Nan Shandong", "China", "5,663,015", "daddy"},
		{"69", "Barcelona", "Spain", "5,658,472", "daddy"},
		{"70", "Saint Petersburg", "Russia", "5,535,556", "daddy"},
		{"71", "Abidjan", "Ivory Coast", "5,515,790", "daddy"},
		{"72", "Yangon", "Myanmar", "5,514,454", "daddy"},
		{"73", "Fukuoka", "Japan", "5,502,591", "daddy"},
		{"74", "Alexandria", "Egypt", "5,483,605", "daddy"},
		{"75", "Guadalajara", "Mexico", "5,339,583", "daddy"},
		{"76", "Ankara", "Turkey", "5,309,690", "daddy"},
		{"77", "Chittagong", "Bangladesh", "5,252,842", "daddy"},
		{"78", "Addis Ababa", "Ethiopia", "5,227,794", "daddy"},
		{"79", "Melbourne", "Australia", "5,150,766", "daddy"},
		{"80", "Nairobi", "Kenya", "5,118,844", "daddy"},
		{"81", "Hanoi", "Vietnam", "5,067,352", "daddy"},
		{"82", "Sydney", "Australia", "5,056,571", "daddy"},
		{"83", "Monterrey", "Mexico", "5,036,535", "daddy"},
		{"84", "Changsha", "China", "4,809,887", "daddy"},
		{"85", "Brasilia", "Brazil", "4,803,877", "daddy"},
		{"86", "Cape Town", "South Africa", "4,800,954", "daddy"},
		{"87", "Jiddah", "Saudi Arabia", "4,780,740", "daddy"},
		{"88", "Urumqi", "China", "4,710,203", "daddy"},
		{"89", "Kunming", "China", "4,657,381", "daddy"},
		{"90", "Changchun", "China", "4,616,002", "daddy"},
		{"91", "Hefei", "China", "4,496,456", "daddy"},
		{"92", "Shantou", "China", "4,490,411", "daddy"},
		{"93", "Xinbei", "Taiwan", "4,470,672", "daddy"},
		{"94", "Kabul", "Afghanistan", "4,457,882", "daddy"},
		{"95", "Ningbo", "China", "4,405,292", "daddy"},
		{"96", "Tel Aviv", "Israel", "4,343,584", "daddy"},
		{"97", "Yaounde", "Cameroon", "4,336,670", "daddy"},
		{"98", "Rome", "Italy", "4,297,877", "daddy"},
		{"99", "Shijiazhuang", "China", "4,285,135", "daddy"},
		{"100", "Montreal", "Canada", "4,276,526", "daddy"},
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
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
