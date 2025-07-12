# Sūtra: The Fabric of Operational Truth



> In any large system or organization, the truth is scattered. Sūtra is the thread that weaves it back together — A fabric of Operational Truth for proactive and reactive control.

**Stop debugging dashboards and logs. Start querying reality.** Sūtra is a decentralized, peer-to-peer "truth-finding machine" built to bring clarity to the chaos of modern infrastructure.


### The Problem: Organizational Brain Fog

Modern systems are a sea of disjoint conflicting data. In a crisis:
*   Your logging platform screams about application errors.
*   Your metrics platform shows a database latency spike.
*   Your CI/CD system reports a deployment failure.
*   Your newly found Autonomous AI agent is confused by the conflicting signals.
*   Your IOT sensors report unexpected behavior.

Ten engineers on a call spend the first 45 minutes simply arguing about which signal is the cause and which is the effect.There is no single, trusted, real-time map of what is actually happening. Even a Literal AI God that lacks context will find an urge to gossip. We are flying essentially blind.

### The Solution: A Decentralized Truth Machine

Sūtra is not another database or dashboard. It's a lightweight, resilient mesh of agents that you run within your infrastructure. These agents constantly consume facts (k-paks — knowledge packets) from various "scouts," reconcile conflicting information using a (currently) deterministic algorithm, and gossip with each other until the entire mesh converges on a single, shared understanding of reality.

It is designed to be the **nervous system for your infrastructure**, providing a clean, queryable, real-time source of truth for your human operators and your AI automation.

### Key Features (CURRENT SUITE)

*   **Radically Simple Deployment:** A single Go binary with no external dependencies. Deploy a powerful distributed system with just a YAML file.
*   **Decentralized & Resilient:** No single point of failure. The mesh is designed to survive node and network outages using a peer-to-peer gossip protocol.
*   **Deterministic Reconciliation:** A simple but powerful `confidence + timestamp` algorithm resolves conflicting facts, ensuring the mesh always converges on the most trustworthy information.
*   **Immutable Audit Trail:** A Write-Ahead Log (WAL) on each agent provides a complete, auditable history of every claim the system has ever processed.
*   **🆕 Time-To-Live (TTL) & Garbage Collection:** Built-in memory management with configurable TTL for k-paks and automatic cleanup of expired data to prevent unbounded memory growth.

### How It Works

1.  **Scouts:** Simple, independent processes that observe the world (a log file, a metric, a K8s event etc.) and report facts (`k-paks`) to the nearest agent. Scouts can be anything from a log parser to a Kubernetes event watcher, or even an AI agent that reports its own findings or a python script that monitors a specific data source and produces knowledge packets.
2.  **Sūtra Agents:** The weavers. They receive facts, gossip & reconcile them against their current beliefs, and write the winner to their local log.
3.  **The Mesh:** Agents gossip their accepted truths to their peers until the entire fabric is in a consistent state.
4.  **Consumers:** Humans (`sutra-ctl`) or AI operators (`sutra-py-sdk` under development) query any agent in the mesh to get the single, reconciled answer.

### Getting Started (Quickstart)

Try out a 3-node Sūtra mesh on your local machine in under 5 minutes. a very primitive taste of how Sūtra works and idea proving the concept.

**1. Prerequisites:**
*   Go (1.19+) installed.
*   At least 3 terminal windows.

**2. Build the Binaries:**
```bash
# Option 1: Use the Makefile (recommended)
make build

# Option 2: Manual build
go build -o bin/sutra-agent.exe ./cmd/sutra-agent
go build -o bin/sutra-ctl.exe ./cmd/sutra-ctl
```

**3. Run the Agents:**
*In Terminal 1 (Bootstrap Node):*
```bash
.\bin\sutra-agent.exe --config configs\agent1.yaml
```

*In Terminal 2:*
```bash
.\bin\sutra-agent.exe --config configs\agent2.yaml
```

*In Terminal 3:*
```bash
.\bin\sutra-agent.exe --config configs\agent3.yaml
```

*The agents will discover each other and form a mesh.*

**4. Interact with the Mesh:**
*In Terminal 4:*
```bash
# Ingest a low-confidence fact into Agent 2 
.\bin\sutra-ctl.exe --agent localhost:9092 ingest "pluto" "is_planet" "true" --source "OldTextbook" --confidence 0.6

# Ingest a conflicting, high-confidence fact into Agent 3 with TTL
.\bin\sutra-ctl.exe --agent localhost:9094 ingest "pluto" "is_planet" "false" --source "IAU-2006" --confidence 0.99 --ttl 300

# Ingest a temporary fact with short TTL (expires in 30 seconds)
.\bin\sutra-ctl.exe --agent localhost:9090 ingest "server1" "status" "maintenance" --source "admin" --ttl 30

# Wait 5 seconds, then query Agent 1 for the converged truth
.\bin\sutra-ctl.exe --agent localhost:9090 query "pluto"
# EXPECTED OUTPUT: The system correctly reports that 'pluto is_planet false'

# Query temporary data
.\bin\sutra-ctl.exe --agent localhost:9090 query "server1"
# EXPECTED OUTPUT: Shows maintenance status (will auto-expire after TTL)

# Verify the mesh formed correctly
.\bin\sutra-ctl.exe --agent localhost:9090 peers
# EXPECTED OUTPUT: Should show 3 connected agents
```


**5. Troubleshooting:**
- **"connection refused" errors**: Make sure all agents are started and wait few seconds for mesh formation
- **Build fails**: Ensure Go 1.19+ is installed and run `go mod tidy` first
- **Agents won't connect**: Check that ports 9090-9095 are not blocked by firewall
- **No output from queries**: Verify agents are running with `.\bin\sutra-ctl.exe health`

**7. Quick Tests:**
```bash
# Test single agent health
.\bin\sutra-ctl.exe --agent localhost:9090 health

# Test mesh connectivity
.\bin\sutra-ctl.exe --agent localhost:9090 peers

# Run comprehensive test suite
make test
```

### The Journey of Sūtra: A Living Roadmap

Sūtra is an ambitious project. Our development is phased to deliver value and stability at every step. Open to Suggestions, criticisms and contributions!

*   **Phase 1: Public Launch & Showcase (Under Development — Launch Done)**
    *   Core `sutra-agent`, gossip mesh, reconciliation logic, and `sutra-ctl`. The minimum viable primitive brain is fully functional. The goal was to prove the core idea works.
    * Begin parallel ideation & development of the `sutra-py-sdk` to enable Python-based (and for possibly more languages) scouts with proper categorisations. Must feel like filling out a form, not writing code to conceive a scout.

*   **Phase 2: The Security Foundation ("Sūtra Secure") (🔜 UP NEXT)**
    *   **Goal:** Make Sūtra enterprise-ready and trustworthy.
    *   **Features:** End-to-end mTLS encryption, cryptographic signatures on all facts, and verifiable source identities.

*   **Phase 3: The Stability Foundation ("Sūtra Stable") ✅ COMPLETED**
    *   **Goal:** Ensure long-term operational health for production infrastructure.
    *   **Features:** ✅ Time-to-Live (TTL) on k-paks, ✅ automatic garbage collection, and WAL compaction. Possible use of Merkle trees truth hash verification resulting in efficient deduplication.

*   **Phase 4: The Expressiveness Foundation ("Sūtra Graph")**
    *   **Goal:** Evolve the data model to represent complex, real-world systems.
    *   **Features:** Support for a Property Graph model (Nodes & Edges giving rise to k-nodes and k-edges) instead of triples, enabling far richer queries and insights.Basic graph query capabilities

*   **Phase 5: The Intelligence Layer ("Sūtra Adaptive")**
    *   **Goal:** Transform Sūtra from a deterministic engine into a smart, adaptive fabric.
    *   **Features:** An adaptive source reputation system that learns to trust and distrust noisy/malicious scouts over time.  Self-tuning trust scores

### How to Contribute

This is a young project with an ambitious vision, and we welcome contributors of all levels. The best way to get started is to help us build the ecosystem.

*   **🐍 Build a Scout:** The `sutra-py-sdk` will make it easy ! Do you have a favorite tool you want to see integrated?
    *   **Good First Idea:** A **Kubernetes Events Scout** that reports on pod failures.
    *   **Good First Idea:** A **Stripe Scout** that reports on business metrics like payment failures.
    *   **Good First Idea:** A **CI/CD Scout** for GitHub Actions or GitLab.

*   **📖 Improve the Docs:** See a section that's confusing? Find a typo? PRs to improve documentation are always welcome.

*   **🤔 Tackle a Core Challenge:** Fascinated by distributed systems and new found power of autonomous agency?
    *   Help research and prototype features for the **Security** or **Stability** milestones.
    *   Read the [DESIGN_DOCS](./docs) folder (once created) and provide feedback on the V2.0 architecture.

Open an issue to discuss your ideas!

### License

Sūtra is licensed under the [Apache License 2.0](./LICENSE).
