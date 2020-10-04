package weatherbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func AppWeatherMentionHandler(w http.ResponseWriter, r *http.Request) {
	// get the request body
	defer func() {
		_ = r.Body.Close()
	}()
	body, _ := ioutil.ReadAll(r.Body)

	// unmarshal the whole body (JSON) into a map
	m := make(map[string]interface{})
	err := json.Unmarshal(body, &m)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error unmarshalling body: %v", err)
		return
	}

	// if it's the first them this server is registered with slack,
	// a challenge code needs to be returned
	//fmt.Fprintf(w, "%s", m["challenge"])

	// extract the event field into another map
	nm := m["event"].(map[string]interface{})

	// get the text field
	text := fmt.Sprintf("%v", nm["text"])
	str := strings.Split(text, "<bot user ID>")
	city := strings.Trim(str[1], " ")
	// get the channel id
	channel := fmt.Sprintf("%v", nm["channel"])
	token := "User's Slack Bot Token"

	// get current weather
	weather, err := getWeather(city)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error calling openweather API: %v", err)
		return
	}

	// send the weather to slack channel
	err = sendMessage(token, channel, weather)
	if err != nil {
		_, _ = fmt.Fprintf(w, "error sending message to Slack: %v", err)
		return
	}
}

func sendMessage(token, channel, text string) error {
	postURL := "https://slack.com/api/chat.postMessage"
	data := url.Values{"token": {token}, "channel": {channel}, "text": {text}}
	_, err := http.PostForm(postURL, data)

	return err
}

func getWeather(sym string) (string, error) {
	appId := "User's App ID"
	// sym is City name || City name, state code || city name, state code, country code
	owUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&units=metric&APPID=%s", sym, appId)
	sym = strings.Title(strings.ToLower(sym))

	resp, err := http.Get(owUrl)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)

	log.Printf("%s\n", string(body))
	m := make(map[string]interface{})
	err = json.Unmarshal(body, &m)
	if err != nil {
		return "", err
	}

	if len(m) == 2 {
		// m = {"cod":"404","message":"city not found"}
		return "*City " + sym + " is not found*", nil
	}

	m1 := m["weather"].([]interface{})
	m2 := m["main"].(map[string]interface{})
	m3 := m["wind"].(map[string]interface{})
	m4 := m1[0].(map[string]interface{})

	emojimap := map[string]string{
		"Clear":        ":sun_with_face: :full_moon_with_face:",
		"Drizzle":      ":partly_sunny: :closed_umbrella:",
		"Clouds":       ":partly_sunny: :cloud:",
		"Rain":         ":umbrella:",
		"Thunderstorm": ":zap:",
		"Snow":         ":snowflake: :snowman:",
		"Mist":         ":foggy:",
		"Haze":         ":foggy:",
		"Smoke":        ":foggy:",
		"Squall":       ":foggy:",
		"Fog":          ":foggy:",
		"Sand":         ":foggy:",
		"Dust":         ":foggy:",
		"Ash":          ":foggy:",
		"Tornado":      ":foggy:"}

	condition := fmt.Sprintf("%s", m4["main"])
	utc := time.Now().UTC().Format(time.ANSIC)

	mtz := m["timezone"].(float64)
	tz := time.Now().UTC().Add(time.Duration(mtz) * time.Second)
	timeIn := tz.Format(time.ANSIC)

	cityName := m["name"].(string)

	s := "*Weather in " + cityName + "* \n" +
		"Condition: _" + condition + "  " + fmt.Sprintf("%s", emojimap[condition]) + "_\n" +
		"Current: " + fmt.Sprintf("%.0f°C", m2["temp"]) +
		"    Low: " + fmt.Sprintf("%.0f°C", m2["temp_min"]) +
		"    High: " + fmt.Sprintf("%.0f°C", m2["temp_max"]) + "\n" +
		"Wind speed: " + fmt.Sprintf("%.1f", m3["speed"]) + "m/s" +
		"  Feels like: " + fmt.Sprintf("%.0f°C", m2["feels_like"]) + "\n" +
		"Humidity: " + fmt.Sprintf("%.0f", m2["humidity"]) + " % \n" +
		"UTC: " + utc + "\n" +
		"Local time: " + timeIn

	return s, nil
}
