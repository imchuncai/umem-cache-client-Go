# SPDX-License-Identifier: BSD-3-Clause
# Copyright (C) 2025, Shu De Zheng <imchuncai@gmail.com>. All Rights Reserved.

src = ../umem-cache

ifdef SRC
	src = $(SRC)
endif

ifndef TLS
	TLS = 0
endif

ifndef RAFT
	RAFT = 0
endif

ifndef DEBUG
	DEBUG = 0
endif

define run-test =
	$(eval test-case := $(shell		\
		if [[ $(2) = 0 ]]; then		\
			echo TestClient;	\
		else				\
			echo TestCluster;	\
		fi))
	
	go test -timeout=1m -failfast -p=1 -v -run=$(test-case) -args $(1) $(3) $(4)
endef

define debug-test =
	$(MAKE) -s -C $(src) RAFT=$(1) TLS=$(2) DEBUG=1 TCP_TIMEOUT=1000
	$(call run-test,$(src)/umem-cache,$(1),$(2),1)
endef

target = test-src
ifdef EXE
	target = test-exe
endif

test: $(target)

test-src:
	$(call debug-test,1,1)
	@echo
	$(call debug-test,1,0)
	@echo
	$(call debug-test,0,1)
	@echo
	$(call debug-test,0,0)
	@echo

	$(MAKE) -s -C $(src) RAFT=1 TLS=1 DEBUG=1 TEST_ELECTION_WITH_UNSTABLE_LOG=1
	go test -timeout=1m -failfast -p=1 -v -run=TestElectionWithUnstableLog -args $(src)/umem-cache 1 1
	@echo

	$(MAKE) -s -C $(src) RAFT=1 TLS=1 DEBUG=1 TEST_ELECTION_WITH_UNSTABLE_GROW_LOG=1
	go test -timeout=1m -failfast -p=1 -v -run=TestElectionWithUnstableGrowLog -args $(src)/umem-cache 1 1
	@echo

	$(MAKE) -s -C $(src) RAFT=1 TLS=1 DEBUG=1 TEST_VOTE_WITH_LOG0=1
	go test -timeout=1m -failfast -p=1 -v -run=TestVoteWithLog0 -args $(src)/umem-cache 1 1

test-exe:
	$(call run-test,$(EXE),$(RAFT),$(TLS),$(DEBUG))

test-client-example:
	go test -timeout=1m -failfast -p=1 -v -run=ExampleClient

test-cluster-example:
	go test -timeout=1m -failfast -p=1 -v -run=ExampleCluster

clean:
	@rm -rf logs

PHONEY: test test-src test-exe clean
