
# Blink, a powerful source-based package manager. Core of ApertureOS.
# Want to use it for your own project?
#	Blink is completely FOSS (Free and Open Source),
#	edit, publish, use, contribute to Blink however you prefer.
#  Copyright (C) 2025 Aperture OS

#  This program is free software: you can redistribute it and/or modify
#	 it under the terms of the GNU General Public License as published by
#  the Free Software Foundation, either version 3 of the License, or
#  (at your option) any later version.

#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.

#  You should have received a copy of the GNU General Public License
#  along with this program.  If not, see <https://www.gnu.org/licenses/>.


APP := main
SRC := ./

all:
	echo "Building: testing binary, giant size."
	CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o $(APP) $(SRC)

release: 
	echo "Building: production-ready binary."
	CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o $(APP) $(SRC)
	echo "Building: optimizing with UPX, really slow."
	upx --best --ultra-brute $(APP)

static: 
	go build -o $(APP) $(SRC)

clean:
	rm -f $(APP)

# i hope whoever is reading this can feel how much fucking time i 
# spent (over 1.5hours) to fix a bug about reading files and directories and whatnot
# because i forgot i used relative paths and not absolute so i was running the binary from 
# / and not /src (/ = the root directory of the project) so it was fucking me up, also
# i hope whenever i had the idea to use absolute path, in another reality/universe
# someone would shoot my in the leg 2 times so i had to go to the hospital and never
# made that decision. well thats all folks, remember to not use relative paths, even for testing!