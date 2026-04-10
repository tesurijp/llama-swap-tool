SUBDIRS := llama-launcher ol-proxy

DEST_DIR := $(CURDIR)/build

.PHONY: all clean $(SUBDIRS)

all: $(SUBDIRS)

$(SUBDIRS):
	@mkdir -p $(DEST_DIR)
	$(MAKE) -C $@
	cp $@/build/* $(DEST_DIR)/ 

clean:
	@for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir clean; \
	done
	rm -rf $(DEST_DIR)
