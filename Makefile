GO =		go
GOBUILDFLAGS =	-v

DESTDIR =
PREFIX =	/usr/local
BINDIR =	${PREFIX}/bin
MANDIR =	${PREFIX}/man

INSTALL =	install
INSTALL_PROGRAM=${INSTALL} -m 0555
INSTALL_MAN =	${INSTALL} -m 0444

all: plakar

plakar:
	${GO} build ${GOBUILDFLAGS} ./cmd/plakar

install:
	mkdir -p ${DESTDIR}${BINDIR}
	mkdir -p ${DESTDIR}${MANDIR}/man1
	${INSTALL_PROGRAM} plakar ${DESTDIR}${BINDIR}
	find cmd/plakar -iname \*.1 -exec \
		${INSTALL_MAN} {} ${DESTDIR}${MANDIR}/man1 \;

.PHONY: all plakar install
