package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type Outcome struct {
    TypeName string  `json:"type_name"`
    Type     string  `json:"type"`
    Line     *string `json:"line"`
    Odds     float64 `json:"odds"`
}

type OddsData struct {
    HomeTeam      string    `json:"home_team"`
    AwayTeam      string    `json:"away_team"`
    Time          int64     `json:"time"`
    EventID       string    `json:"event_id"`
    League        string    `json:"league"`
    Sport         string    `json:"sport"`
    CurrentMinute int       `json:"current_minute"`
    Outcomes      []Outcome `json:"outcomes"`
}

type FormattedData struct {
    MatchName     string                 `json:"match_name"`
    Time          string                 `json:"time"`
    EventID       string                 `json:"event_id"`
    League        string                 `json:"league"`
    Sport         string                 `json:"sport"`
    CurrentMinute int                    `json:"current_minute"`
    FormattedData map[string]map[string][]string `json:"formatted_data"`
}

func formatOddsData(data OddsData) (FormattedData, error) {
    formattedData := FormattedData{
        MatchName:     fmt.Sprintf("%s vs %s", data.HomeTeam, data.AwayTeam),
        Time:          time.Unix(data.Time, 0).Format("2006-01-02 15:04:05"),
        EventID:       data.EventID,
        League:        data.League,
        Sport:         data.Sport,
        CurrentMinute: data.CurrentMinute,
        FormattedData: map[string]map[string][]string{
            "Match": {},
            "1H":    {},
            "2H":    {},
        },
    }

    for _, outcome := range data.Outcomes {
        period := "Match"
        if strings.HasPrefix(outcome.TypeName, "1H") {
            period = "1H"
        } else if strings.HasPrefix(outcome.TypeName, "2H") {
            period = "2H"
        }

        betType := strings.TrimPrefix(strings.TrimPrefix(outcome.TypeName, "1H"), "2H")
        formattedOutcome := fmt.Sprintf("%s: %s @ %.2f", outcome.Type, getLine(outcome.Line), outcome.Odds)

        if _, exists := formattedData.FormattedData[period][betType]; !exists {
            formattedData.FormattedData[period][betType] = []string{}
        }
        formattedData.FormattedData[period][betType] = append(formattedData.FormattedData[period][betType], formattedOutcome)
    }

    return formattedData, nil
}

func getLine(line *string) string {
    if line == nil {
        return "N/A"
    }
    return *line
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    files, err := ioutil.ReadDir("/odds_data")
    if err != nil {
        http.Error(w, "Failed to read directory", http.StatusInternalServerError)
        return
    }

    var fileList []string
    for _, file := range files {
        if filepath.Ext(file.Name()) == ".jsonl" {
            fileList = append(fileList, file.Name())
        }
    }

    tmpl, err := template.New("home").Parse(homeTemplate)
    if err != nil {
        http.Error(w, "Failed to parse template", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, fileList)
}

func getOddsHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("filename")
    if filename == "" {
        http.Error(w, "Filename not specified", http.StatusBadRequest)
        return
    }

    tmpl, err := template.New("odds").Parse(oddsTemplate)
    if err != nil {
        http.Error(w, "Failed to parse template", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, filename)
}

func getLastLineHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("filename")
    if filename == "" {
        http.Error(w, "Filename not specified", http.StatusBadRequest)
        return
    }

    filePath := filepath.Join("/odds_data", filename)
    file, err := os.Open(filePath)
    if err != nil {
        http.Error(w, fmt.Sprintf("File not found: %s", filename), http.StatusNotFound)
        return
    }
    defer file.Close()

    lines, err := ioutil.ReadAll(file)
    if err != nil || len(lines) == 0 {
        http.Error(w, "File is empty", http.StatusNotFound)
        return
    }

    lastLine := strings.TrimSpace(string(lines))
    var data OddsData
    if err := json.Unmarshal([]byte(lastLine), &data); err != nil {
        http.Error(w, "Invalid JSON in the last line", http.StatusBadRequest)
        return
    }

    formattedData, err := formatOddsData(data)
    if err != nil {
        http.Error(w, "Error processing data", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(formattedData)
}

const homeTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Odds Data Home</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; padding: 20px; max-width: 800px; margin: 0 auto; }
        h1 { color: #333; }
        ul { list-style-type: none; padding: 0; }
        li { margin-bottom: 10px; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <h1>Odds Data Files</h1>
    <ul>
    {{range .}}
        <li><a href="/get_odds?filename={{.}}">{{.}}</a></li>
    {{end}}
    </ul>
</body>
</html>
`

const oddsTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Match Data</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; padding: 20px; max-width: 800px; margin: 0 auto; }
        h1 { color: #333; }
        h2 { color: #666; }
        .period { margin-bottom: 20px; }
        .bet-type { margin-bottom: 10px; }
        .outcomes { display: flex; flex-wrap: wrap; }
        .outcome { background-color: #f0f0f0; padding: 5px 10px; margin: 5px; border-radius: 5px; }
        .back-link { margin-top: 20px; }
        #error-message { color: red; }
    </style>
</head>
<body>
    <h1 id="match-name"></h1>
    <p id="event-info"></p>
    <div id="odds-data"></div>
    <p id="error-message"></p>
    <div class="back-link">
        <a href="/">Back to file list</a>
    </div>

     <script>
        function updateOdds() {
            fetch('/get_last_line?filename=' + encodeURIComponent('{{ filename }}'))
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('error-message').textContent = data.error;
                        return;
                    }
                    document.getElementById('error-message').textContent = '';
                    document.getElementById('match-name').textContent = data.match_name;
                    document.getElementById('event-info').textContent = 'Event ID: ${data.event_id} | Time: ${data.time} | League: ${data.league} | Sport: ${data.sport} | Current Minute: ${data.current_minute}';

                    let oddsHtml = '';
                    for (const [period, types] of Object.entries(data.formatted_data)) {
                        if (Object.keys(types).length > 0) {
                            oddsHtml += '<div class="period"><h2>${period}</h2>';
                            for (const [betType, outcomes] of Object.entries(types)) {
                                oddsHtml += '<div class="bet-type"><h3>${betType}</h3><div class="outcomes">';
                                for (const outcome of outcomes) {
                                    oddsHtml += '<span class="outcome">${outcome}</span>';
                                }
                                oddsHtml += '</div></div>';
                            }
                            oddsHtml += '</div>';
                        }
                    }
                    document.getElementById('odds-data').innerHTML = oddsHtml;
                })
                .catch(error => {
                    console.error('Error:', error);
                    document.getElementById('error-message').textContent = 'Failed to fetch data. Please try again.';
                });
        }

        updateOdds();
        setInterval(updateOdds, 500);
        </script>
</body>
</html>
`

func main() {
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/get_odds", getOddsHandler)
    http.HandleFunc("/get_last_line", getLastLineHandler)

    log.Println("Server started at :8002")
    log.Fatal(http.ListenAndServe(":8002", nil))
}