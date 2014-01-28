
package main

import (
  "fmt"
  "encoding/json"
  "io/ioutil"
  "net/http"
  "github.com/ajstarks/svgo"
  "image/gif"
  "image/png"
  "image"
  "image/color"
  "math/rand"
  "time"
  "runtime"
)

// Data structure to unmarshal body of POST request
type GameSettings struct {
  W int // width of the playground
  H int // height of the playground
  Generations int // only for benchmark: number of generation we like to calculate
  ImageType string // only for visual output: the format of the returned image
  Concurrency int // Number of threads
  Benchmark bool // enable or disable benchmakr mode
  Life int // value for randon life generation
}

// handles all incoming API requests
func ApiHandler(wr http.ResponseWriter, r *http.Request){
  var gameId string // unique id if a game instance
  c, _ := r.Cookie("GAME_ID") // try to read browser cookie
  if(c==nil){
    // if there is no cookie found, set one
    gameId = fmt.Sprint(rand.Intn(10000))
    cookie := http.Cookie{Name: "GAME_ID", Value: gameId}
    http.SetCookie(wr, &cookie)
  } else {
    // if there is a cookie, use it as gameId
    gameIdStr := fmt.Sprint(c)
    gameId = gameIdStr[len("GAME_ID="):]
  }

  // must be the request to create a new game
  if(r.Method == "POST"){
    var gs GameSettings //create variable of structure

    // read request body
    body, _ := ioutil.ReadAll(r.Body) // TODO: error handling

    // unmarshal body into variable of GameSettings
    json.Unmarshal(body, &gs) // TODO: error handling

    // create a new game with the given settings
    game := createGame(gs)

    // store the game in a map with gameId as key
    games[gameId] = game

  }
  // request to get the current game state and trigger the next round
  if(r.Method == "GET"){
    // parse nmber of genartion from URL
    gen := r.URL.Path[len("/api/"):]

    // load the game that matchs the ID from the request
    var game Game
    game = games[gameId]

    if(game.settings.Benchmark == false) {
      if(gen == "new"){
        // serve a load icon
        http.Redirect(wr,r, "/img/load.gif", 302)
      } else {
        // do not calulate the next generation, for the first get request of a game
        if(gen != "0"){
          // trigger the next generation calulation
          game.nextGen()
        }
        // store the results
        games[gameId] = game

        // draw the current state and print it to the response writter
        game.draw(wr)
      }
    } else {
      game.benchmark(wr)
    }
  }
}

func main() {
  // number of thrads should not be higher than the number of CPUs
  runtime.GOMAXPROCS(runtime.NumCPU())

  // static file server to serve the forntend
  http.Handle("/", http.FileServer(http.Dir("./assets/")))
  // handle all API requests
  http.HandleFunc("/api/", ApiHandler)

  // tell to world: I`am living
  fmt.Println("Game is up and running")

  // let the server run on port 8080 and listen to everbody
  http.ListenAndServe(":8080", nil)
}

/*
  GAME
*/

// here we store the metrics that are needed for a benchmark
type GameMetrics struct {
  generations int
  avgGenTime map[int64]float32
}

// data structure of a game with setting and metrics
type Game struct {
  settings GameSettings
  playground [][]bool
  metrics GameMetrics
}

// create a map to hold all games in memory
var games = make(map[string]Game)

// game factory
func createGame(gs GameSettings) Game {
  var g Game
  g.metrics.avgGenTime = make(map[int64]float32)
  g.metrics.generations = 0
  g.settings = gs
  // create a field to play
  g.playground = g.createPlayground(g.settings.W, g.settings.H)
  // make some life
  g.rndFill()

  return g
}

// create a 2d array from type bool as playground
func (g *Game) createPlayground(w, h int) [][]bool {
  field := make([][]bool, w)
  for i := range  field {
    field[i] = make([]bool,h)
  }
 return field
}

// iterate over all cell and give them randomly life
func (g *Game) rndFill() {
  for i := 0; i < g.settings.W; i++ {
    for j := 0; j < g.settings.H; j++ {
      // if g.settings.Life is high, the probability that the cell will life is low
      rnd := rand.Intn(g.settings.Life)
      state := false
      if(rnd==1){state=true}

      g.playground[i][j] = state
    }
  }
}

func (g *Game) nextGen() {
  // save the time of the function start
  sTime := time.Now().UnixNano()/1e6 // milliseconds

  // calulate the slice size for one thread
  sliceSize := g.settings.W / g.settings.Concurrency

  // array to store the thread channels
  loops := make([]chan [][]bool, g.settings.Concurrency )

  for thread :=0; thread < g.settings.Concurrency ; thread++ {
    // create a new channel
    c := make(chan [][]bool)
    // run a working thread
    go g.worker(sliceSize*thread, sliceSize*(thread+1), g.playground, c)
    loops[thread] = c
  }

  // pickup the resaults of all channels and recompose them
  for thread, c := range loops {
    copy(g.playground[sliceSize*thread:sliceSize*(thread+1)],<-c)
  }

  // store the of the function
  eTime := time.Now().UnixNano()/1e6 // milliseconds
  // get the current time with the strictness of a hundredth second
  cTime := time.Now().UnixNano()/1e8 // milliseconds / 100

  // calulate the benchmark metrics
  if (g.metrics.avgGenTime[cTime]!=0) {
    g.metrics.avgGenTime[cTime] = (g.metrics.avgGenTime[cTime] + float32(eTime - sTime))/2
  } else{
     g.metrics.avgGenTime[cTime] = float32(eTime - sTime)
  }
}

// the guy for the heavy stuff
func (g *Game) worker(sW int, eW int, pg [][]bool, c chan [][]bool) {
  sliceW := eW - sW // width of the current slice
  sliceH := len(pg[0]) // height of the current slice

  // a new empty playground to store the resault of the slice
  field := g.createPlayground(sliceW, sliceH)

  for i := 0; i < sliceW; i++ {
    for j := 0; j < sliceH; j++ {
      // check the environment of each cell and decide if they can life there
      field[i][j] = g.checkEnv(i+sW, j, pg)
    }
  }
  // push the resoult to channel
  c <- field
}

func (g *Game) checkEnv(x int ,y int, pg [][]bool) bool {
  alive := 0

  // count all living cells in a area 3x3 around a cell
  for i := -1; i <= 1; i++ {
    for j := -1; j <= 1; j++ {
      a := x+i
      b := y+j

      // first check if the cell are on the playground
      if(a>=0 && a<len(pg)-1 && b>=0 && b<len(pg[0])-1){
        if(pg[a][b]==true){
          // increment if they live
          alive++
        }
      }
    }
  }
  // true if there are 3 living cells in this area
  // or if there are 4, the cell in the middle should also live
  return alive == 3 || (pg[x][y]==true && alive == 4)
}

// just send the image in the requested format
func (g *Game) draw(wr http.ResponseWriter) {
  w := len(g.playground)
  h := len(g.playground[0])

  if (g.settings.ImageType!="svg") {
    // creates an empty image
    m := image.NewRGBA(image.Rect(0, 0, w, h))

    for i := 0; i < w; i++ {
      for j := 0; j < h; j++ {
        if(g.playground[i][j]==true){
          m.Set(i, j, color.RGBA{0, 0, 0, 255})
        } else {
          m.Set(i, j, color.RGBA{255, 255, 255, 255})
        }
      }
    }

    if (g.settings.ImageType == "gif") {gif.Encode(wr, m, nil)}
    if (g.settings.ImageType == "png") {png.Encode(wr, m)}
  } else {
    wr.Header().Set("Content-Type", "image/svg+xml")
    s := svg.New(wr)
    s.Start(w, h)

    for i := 0; i < w; i++ {
      for j := 0; j <h; j++ {
        if(g.playground[i][j]==true){
          s.Rect(i, j, 1, 1)
        }
      }
    }
    s.End()
  }
}

func (g *Game) benchmark(w http.ResponseWriter) {
  // reset
  g.metrics.avgGenTime = make(map[int64]float32)
  // store the start time of the benchmark
  sTime:= time.Now().UnixNano()/1e6

  // simulate n-generations
  for i := 0; i < g.settings.Generations; i++ {
    g.nextGen()
  }
  // parse the metric resaults as JSON string
  jString := "{\"gphms\":["
  for key, value := range g.metrics.avgGenTime {
    if(jString != "{\"gphms\":[" ) {jString += ","}
    jString += "{\"date\": " + fmt.Sprint(key) + ", \"value\": " + fmt.Sprint(value) + "}"
  }
  duration := time.Now().UnixNano()/1e6 - sTime
  jString += "], \"duration\":" + fmt.Sprint(duration)  + "}"

  // send it as JSON
  w.Header().Set("Content-Type", "application/json")
  fmt.Fprint(w, jString)
}





