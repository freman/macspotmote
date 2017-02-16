# MacSpotMote

Control Spotify on OSX via a web interface - a hax for communal ability to pause, skip and control the volume.

## Installation

Make sure you have go installed

```
go get github.com/freman/macspotmote
$GOPATH/bin/macspotmote
```

## Modifying the index.html file

If you're going to modify the index.html file you're going to need rice to compile it in

```
cd $GOPATH/src/github.com/freman/macspotmote
go get github.com/GeertJohan/go.rice/rice
rice go-embed
go build
```

## License

Copyright (c) 2017 Shannon Wynter. Licensed under GPL3. See the [LICENSE.md](LICENSE.md) file for a copy of the license.