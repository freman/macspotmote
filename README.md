# MacSpotMote

Control Spotify on OSX via a web interface - a hax for communal ability to pause, skip and control the volume.

## Installation

Make sure you have go installed

```
go get github.com/freman/macspotmote
$GOPATH/bin/macspotmote
```

## OSX Hax

Wrap the app in [Fluid](http://fluidapp.com/) for a more desktop feel.

![image](https://cloud.githubusercontent.com/assets/506680/23049453/f557cfee-f507-11e6-890b-a08819ca40bd.png)


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
