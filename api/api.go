package api

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/bmizerany/pat"
	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/api/robeaux"
)

// Optional restful API through Gobot has access
// all the robots.
type api struct {
	gobot    *gobot.Gobot
	server   *pat.PatternServeMux
	Host     string
	Port     string
	Username string
	Password string
	Cert     string
	Key      string
	handlers []func(http.ResponseWriter, *http.Request)
	start    func(*api)
}

func NewAPI(g *gobot.Gobot) *api {
	return &api{
		gobot: g,
		Port:  "3000",
		start: func(a *api) {
			log.Println("Initializing API on " + a.Host + ":" + a.Port + "...")
			http.Handle("/", a.server)

			go func() {
				if a.Cert != "" && a.Key != "" {
					http.ListenAndServeTLS(a.Host+":"+a.Port, a.Cert, a.Key, nil)
				} else {
					log.Println("WARNING: API using insecure connection. " +
						"We recommend using an SSL certificate with Gobot.")
					http.ListenAndServe(a.Host+":"+a.Port, nil)
				}
			}()
		},
	}
}

func (a *api) AddHandler(f func(http.ResponseWriter, *http.Request)) {
	a.handlers = append(a.handlers, f)
}

// start starts the api using the start function
// sets on the API on initialization.
func (a *api) Start() {
	a.server = pat.New()

	mcpCommandRoute := "/commands/:command"
	deviceCommandRoute := "/robots/:robot/devices/:device/commands/:command"
	robotCommandRoute := "/robots/:robot/commands/:command"

	a.server.Get("/", a.run(a.mcp))
	a.server.Get("/commands", a.run(a.mcpCommands))
	a.server.Get(mcpCommandRoute, a.run(a.executeMcpCommand))
	a.server.Post(mcpCommandRoute, a.run(a.executeMcpCommand))
	a.server.Get("/robots", a.run(a.robots))
	a.server.Get("/robots/:robot", a.run(a.robot))
	a.server.Get("/robots/:robot/commands", a.run(a.robotCommands))
	a.server.Get(robotCommandRoute, a.run(a.executeRobotCommand))
	a.server.Post(robotCommandRoute, a.run(a.executeRobotCommand))
	a.server.Get("/robots/:robot/devices", a.run(a.robotDevices))
	a.server.Get("/robots/:robot/devices/:device", a.run(a.robotDevice))
	a.server.Get("/robots/:robot/devices/:device/commands",
		a.run(a.robotDeviceCommands),
	)
	a.server.Get(deviceCommandRoute, a.run(a.executeDeviceCommand))
	a.server.Post(deviceCommandRoute, a.run(a.executeDeviceCommand))
	a.server.Get("/robots/:robot/connections", a.run(a.robotConnections))
	a.server.Get("/robots/:robot/connections/:connection",
		a.run(a.robotConnection),
	)
	a.server.Get("/:a", a.run(a.robeaux))
	a.server.Get("/:a/", a.run(a.robeaux))
	a.server.Get("/:a/:b", a.run(a.robeaux))
	a.server.Get("/:a/:b/", a.run(a.robeaux))
	a.server.Get("/:a/:b/:c", a.run(a.robeaux))
	a.server.Get("/:a/:b/:c/", a.run(a.robeaux))

	a.start(a)
}

func (a *api) run(f func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		for _, handler := range a.handlers {
			handler(res, req)
		}
		f(res, req)
	}
}

func (a *api) robeaux(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	buf, err := robeaux.Asset(path[1:])
	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
	t := strings.Split(path, ".")
	if t[len(t)-1] == "js" {
		res.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else if t[len(t)-1] == "css" {
		res.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	res.Write(buf)
}

func (a *api) mcp(res http.ResponseWriter, req *http.Request) {
	data, _ := json.Marshal(a.gobot.ToJSON())
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) mcpCommands(res http.ResponseWriter, req *http.Request) {
	data, _ := json.Marshal(a.gobot.ToJSON().Commands)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robots(res http.ResponseWriter, req *http.Request) {
	jsonRobots := []*gobot.JSONRobot{}
	a.gobot.Robots().Each(func(r *gobot.Robot) {
		jsonRobots = append(jsonRobots, r.ToJSON())
	})
	data, _ := json.Marshal(jsonRobots)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robot(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")

	data, _ := json.Marshal(a.gobot.Robot(robot).ToJSON())
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotCommands(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")

	data, _ := json.Marshal(a.gobot.Robot(robot).ToJSON().Commands)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotDevices(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")

	jsonDevices := []*gobot.JSONDevice{}
	a.gobot.Robot(robot).Devices().Each(func(d gobot.Device) {
		jsonDevices = append(jsonDevices, d.ToJSON())
	})
	data, _ := json.Marshal(jsonDevices)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotDevice(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")
	device := req.URL.Query().Get(":device")

	data, _ := json.Marshal(a.gobot.Robot(robot).Device(device).ToJSON())
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotDeviceCommands(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")
	device := req.URL.Query().Get(":device")

	data, _ := json.Marshal(a.gobot.Robot(robot).Device(device).ToJSON().Commands)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotConnections(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")

	jsonConnections := []*gobot.JSONConnection{}
	a.gobot.Robot(robot).Connections().Each(func(c gobot.Connection) {
		jsonConnections = append(jsonConnections, c.ToJSON())
	})
	data, _ := json.Marshal(jsonConnections)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) robotConnection(res http.ResponseWriter, req *http.Request) {
	robot := req.URL.Query().Get(":robot")
	connection := req.URL.Query().Get(":connection")

	data, _ := json.Marshal(a.gobot.Robot(robot).Connection(connection).ToJSON())
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) executeMcpCommand(res http.ResponseWriter, req *http.Request) {
	var data []byte
	body := make(map[string]interface{})
	command := req.URL.Query().Get(":command")

	json.NewDecoder(req.Body).Decode(&body)
	f := a.gobot.Command(command)

	if f != nil {
		data, _ = json.Marshal(f(body))
	} else {
		data, _ = json.Marshal("Unknown Command")
	}

	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) executeDeviceCommand(res http.ResponseWriter, req *http.Request) {
	var data []byte
	robot := req.URL.Query().Get(":robot")
	device := req.URL.Query().Get(":device")
	command := req.URL.Query().Get(":command")
	body := make(map[string]interface{})

	json.NewDecoder(req.Body).Decode(&body)
	d := a.gobot.Robot(robot).Device(device)
	body["robot"] = robot
	f := d.Command(command)

	if f != nil {
		data, _ = json.Marshal(f(body))
	} else {
		data, _ = json.Marshal("Unknown Command")
	}

	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) executeRobotCommand(res http.ResponseWriter, req *http.Request) {
	var data []byte

	robot := req.URL.Query().Get(":robot")
	command := req.URL.Query().Get(":command")

	body := make(map[string]interface{})
	json.NewDecoder(req.Body).Decode(&body)
	r := a.gobot.Robot(robot)
	body["robot"] = robot
	f := r.Command(command)

	if f != nil {
		data, _ = json.Marshal(f(body))
	} else {
		data, _ = json.Marshal("Unknown Command")
	}

	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.Write(data)
}

func (a *api) SetBasicAuth(user, password string) {
	a.Username = user
	a.Password = password
	a.AddHandler(a.basicAuth)
}

func (a *api) SetDebug() {
	a.AddHandler(func(res http.ResponseWriter, req *http.Request) {
		log.Println(req)
	})
}

// basic auth inspired by
// https://github.com/codegangsta/martini-contrib/blob/master/auth/
func (a *api) basicAuth(res http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Authorization")
	if !a.secureCompare(auth,
		"Basic "+base64.StdEncoding.EncodeToString([]byte(a.Username+":"+a.Password)),
	) {
		res.Header().Set("WWW-Authenticate",
			"Basic realm=\"Authorization Required\"",
		)
		http.Error(res, "Not Authorized", http.StatusUnauthorized)
	}
}

func (a *api) secureCompare(given string, actual string) bool {
	if subtle.ConstantTimeEq(int32(len(given)), int32(len(actual))) == 1 {
		return subtle.ConstantTimeCompare([]byte(given), []byte(actual)) == 1
	}
	// Securely compare actual to itself to keep constant time,
	// but always return false
	return subtle.ConstantTimeCompare([]byte(actual), []byte(actual)) == 1 && false
}