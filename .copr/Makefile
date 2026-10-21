# COPR build script: generates SRPM from current git tree
VERSION ?= 0.1.0
outdir ?= .
spec ?= packaging/copr/whatsdown.spec

srpm:
	mkdir -p build_srpm $(outdir)
	if command -v git >/dev/null 2>&1; then \
		git archive --format=tar.gz --prefix=WhatsDown-$(VERSION)/ -o build_srpm/whatsdown-$(VERSION).tar.gz HEAD; \
	else \
		tar --exclude-vcs --exclude=build_srpm --exclude=.git --transform 's,^\.,WhatsDown-$(VERSION),' -czf build_srpm/whatsdown-$(VERSION).tar.gz .; \
	fi
	rpmbuild -bs \
		--define "_sourcedir $(CURDIR)/build_srpm" \
		--define "_srcrpmdir $(outdir)" \
		$(spec)
	rm -rf build_srpm
