###
#   Init
###

mainApp = angular.module('main', [])

mainApp.directive "loaded", ->
  (scope, element, attrs) ->
    element.bind "load", ->
      scope.$apply attrs.loaded

###
#   Controllers
###

### Playground ###


mainApp.controller 'gameCtrl', ($scope, $http, $interval) ->
  $scope.stats = {}
  $scope.intervalId = null
  $scope.imgSize = {}
  graph = new links.Graph document.getElementById 'benchmark'
  graphDataset = []

  graphOptions =
    width: '100%'
    height: "400px"
    moveable: false
    zoomable: false
    vStart: 0
    vEnd: 400
    start: 0
    end: 4000

  _resetStats = ->
    $scope.stats =
      genCount: 0
      prevGenCount: 0
      gps: 0
      avgGps: 0
      runtime: 0

  $scope.settings =
    w: 280
    h: 140
    generations: 100
    imageType: 'svg'
    concurrency: 1
    benchmark: false
    life: 5

  _setImageUrl = ->
    $scope.imgageUrl = "/api/#{$scope.stats.genCount++}"

  _init = () ->
    $interval.cancel $scope.intervalId
    $scope.intervalId = null
    $scope.load = false

    ratio = $scope.settings.w / document.getElementById('container').offsetWidth * 1.1
    $scope.imgSize.w = $scope.settings.w * (1 / ratio)
    $scope.imgSize.h = $scope.settings.h * (1 / ratio)

    $scope.state = 'pause'
    $scope.imgageUrl = "/api/new"

    copyOfSettings = JSON.parse JSON.stringify $scope.settings

    copyOfSettings.w = Number copyOfSettings.w
    copyOfSettings.h = Number copyOfSettings.h
    copyOfSettings.generations = Number copyOfSettings.generations
    copyOfSettings.life = 12 - Number copyOfSettings.life
    copyOfSettings.concurrency = Number copyOfSettings.concurrency

    $http.post("/api/set", copyOfSettings).then ->
      unless $scope.settings.benchmark
        _resetStats()
        _setImageUrl()
      else
        graph.draw graphDataset, graphOptions

  _benchmark = () ->
    $http.get('/api/0').success (data) ->
      $scope.load = false
      c = 0

      for d in data.gphms
        d['date'] = c++ *100

      pluralThread = if $scope.settings.concurrency is 1 then '' else 's'
      graphDataset.push
        label: "#{$scope.settings.w} x #{$scope.settings.h} - #{$scope.settings.concurrency} Thread#{pluralThread} - #{$scope.settings.generations} Gen."
        data: data.gphms

      graph.draw graphDataset, graphOptions


  _startInterval = ->
    $scope.intervalId  = $interval (->
      s = $scope.stats

      s.gps = s.genCount - s.prevGenCount

      s.runtime++
      s.avgGps = Math.round(s.genCount / s.runtime * 100) / 100
      s.prevGenCount = s.genCount
    ), 1000

  $scope.apply = () -> _init()
  $scope.next = () -> _setImageUrl()
  $scope.pause = () ->
    $interval.cancel $scope.intervalId
    $scope.state='pause'
    $scope.load = false

  $scope.stop = () ->
    $scope.load = false
    $scope.state = 'pause'
    _init()
    graphDataset = []
    graph.draw graphDataset, graphOptions

  $scope.play = () ->
    $scope.load = true
    if $scope.settings.benchmark
      _benchmark()
    else
      $scope.state = 'play'
      _startInterval()
      _setImageUrl()

  $scope.genLoaded = ->
    _setImageUrl() if $scope.state is 'play'

  _init()

  $scope.$watch 'settings', _init , true












