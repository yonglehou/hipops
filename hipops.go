package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	flag "github.com/dotcloud/docker/pkg/mflag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

type Configuration struct {
	Apps      []App
	Env       string
	Id        string
	Playbooks []Playbook
	Servers   []Server
}
type App struct {
	Branch string
	Config string
	Data   string
	Host   string
	Image  string
	Name   string
	Repo   string
	Ports  []int
	Start  string
	SshKey string
	Type   string
}

type Server struct {
	Apps []string
	Role string
	Type string
}
type Playbook struct {
	Actions   []DockerAction
	App       string
	Inventory string
	Name      string
	Play      string
	State     string
}
type DockerAction struct {
	Image  string
	Params string
}

func main() {
	var (
		flHosts      = flag.String([]string{"h", "-hosts"}, "", "Inventory Hosts Target e.g. local,aws")
		flConfigFile = flag.String([]string{"c", "-config"}, "./config.json", ".json configuration")
		flPlaybooks  = flag.String([]string{"p", "-playbook-path"}, "../../playbooks/", "Playbooks Path")
		flPrivateKey = flag.String([]string{"k", "-private-key"}, "", "SSH Private Key")
	)
	flag.Parse()
	if *flHosts == "" {
		log.Fatal("Usage: [-h <Inventory Hosts Target>][-k <SSH private key>]")
	}
	config, err := ioutil.ReadFile(*flConfigFile)
	check(err)
	var c Configuration
	err = json.Unmarshal(config, &c)
	check(err)
	for k, v := range c.Apps {
		c.Apps[k].Name = parse(v.Name, c, "")
		c.Apps[k].Data = parse(v.Data, c, "")
	}
	for _, v := range c.Playbooks {
		//fmt.Println(v.App)
		//fmt.Println(parse(v.Name, c, v.App))
		//fmt.Println(parse(v.Actions[0].Params, c, v.App))
		RunCmd("ansible-playbook",
			fmt.Sprintf("%s%s", *flPlaybooks, v.Play),
			"-i", *flHosts,
			"--private-key", *flPrivateKey,
			"-e", fmt.Sprintf("inventory=%s name=%s image=%s state=%s params=\"%s\" repo=%s sshKey=%s branch=%s path=%s",
				parse(v.Inventory, c, v.App),
				parse(v.Name, c, v.App),
				parse(v.Actions[0].Image, c, v.App),
				v.State,
				parse(v.Actions[0].Params, c, v.App),
				parse("{{.App.Repo}}", c, v.App),
				parse("{{.App.SshKey}}", c, v.App),
				parse("{{.App.Branch}}", c, v.App),
				parse("{{.App.Data}}", c, v.App),
			),
			"-vvvvv",
		)
	}

	fmt.Println(flPlaybooks)
	RunCmd("ls", "-l")
}
func format(input string, app string) string {
	app = strings.Replace(app, "{{", "(", -1)
	app = strings.Replace(app, "}}", ")", -1)
	var re = regexp.MustCompile(`({{.App(.&}})*)`)
	input = re.ReplaceAllString(input, fmt.Sprintf("{{%s${2}", app))
	return input
}
func parse(input string, base interface{}, app string) string {
	t := template.New("")
	if app != "" {
		input = format(input, app)
	}
	t, _ = t.Parse(input)
	buf := new(bytes.Buffer)
	t.Execute(buf, base)
	return buf.String()
}
func evaluate(obj Configuration) {
	s := reflect.ValueOf(&obj).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fmt.Printf("%d: %s %s =  %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
}
func check(err error) {
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
}
func RunCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.StdoutPipe()
	check(err)
	stderr, err := cmd.StderrPipe()
	check(err)
	err = cmd.Start()
	check(err)
	defer cmd.Wait()
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
}