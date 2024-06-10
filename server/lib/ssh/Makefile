SHELL = /bin/bash

.PHONY: all

all: build

build:
	@rm -rf milkyway/ cb-milkyway*
	@git clone https://github.com/cloud-barista/cb-milkyway.git -b master milkyway
	@cd milkyway && git reset --hard 092b84065870087769e5f40f30fbdcfe22b372bb && cd ..
	@patch -p1 < milkyway.patch
	@"$(MAKE)" -C milkyway/src
	@cp milkyway/src/cb-milkyway ./sourceFiles/milkyway
