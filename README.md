# Engineering Gotchas: What You Didn’t Think To Ask

Welcome to the **Engineering Gotchas** series, where I look into often-overlooked challenges in system design, distributed processing, and more. Each episode will focus on a particular scenario, with a code implementation in Go (Golang), along with explanations and gotchas that may not be immediately obvious.

## 📚 Table of Contents

- [Overview](#overview)
- [Episodes](#episodes)
  - [Episode 1: Ensuring Fairness in Asynchronous Processing](#episode-1-ensuring-fairness-in-asynchronous-processing)
  - [Episode 2: Implementing Rate Limiting Across Multiple Servers](#episode-2-implementing-rate-limiting-across-multiple-servers)
- [How to Navigate the Series](#how-to-navigate-the-series)
- [Contributing](#contributing)

---

## Overview

In this repository, you'll find solutions to real-world engineering challenges that come up frequently but are often overlooked. Each episode covers a scenario with a common solution and highlights a "gotcha" that you may not have considered. All solutions are written in **Golang**, and while simplified for clarity, they are practical enough to be applied in real-world applications.

Feel free to clone the repository and explore the code. Solutions include detailed comments to make understanding the implementation easier.

## Episodes

### Episode 1: Ensuring Fairness in Asynchronous Processing

**Scenario**:  
You’re building a system to process large transaction records for multiple clients, each submitting up to 1 million records at any given time. The system needs to handle these transactions efficiently, ensuring each client’s submission is processed in a fair and orderly manner, even with multiple clients submitting at the same time.

An initial approach might involve using an **asynchronous process** where a background worker processes transactions one by one, with retry logic in case of failure. However, there’s a challenge:

❓ **How do you ensure that the client who submits their records first is processed first?**  
With an asynchronous approach, there's no inherent guarantee that requests will be processed in the order they're received, especially when multiple workers are processing transactions concurrently.

**Trick**:  
To address this, This episode involves the design of a system where transaction batches are submitted to a queue and processed in the order they arrive. Each client is assigned a lock (mutex) to ensure only one worker processes their transactions at a time, maintaining consistency and preventing overlap.

**The solution includes**:
- Transaction queues to handle batch submissions in order.
- Client-specific locking to ensure that only one worker processes a client’s transactions at a time.
- Retry logic to gracefully handle failures.

🔑 **Follow-up**: What happens when multiple clients are submitting large files simultaneously? How do you ensure no client is unfairly delayed or prioritized, and how do you balance the load across workers?

📂 [Link to Episode 1 Code](./ep1)


### Episode 2: Implementing Rate Limiting Across Multiple Servers

**Scenario**:  
You're tasked with implementing rate limiting in a distributed API system. Your API is behind a load balancer, and incoming requests are routed to multiple servers. The goal is to limit the number of requests a user can make within a specific time frame (say 100 requests per minute). How do you ensure that the rate limit is correctly enforced across all servers?

An initial approach might involve implementing rate limiting on each server individually. However, this leads to a challenge:

❓ **How do you maintain a consistent rate limit across all servers when each server has its own memory and state?**  
With each server enforcing its own rate limit, a user could potentially make the maximum allowed number of requests to each server, effectively bypassing the intended global limit.

**Trick**:  
In this episode, I implemented a simple rate-limiting system that shares state across all servers. By using a central data store (a map was used in the code), we can maintain consistent rate-limiting counters that all servers access and update in real-time.

**The solution includes**:
- Centralized Counter: Use of a map to store and update the request counts for each user
- Atomic Operations: Utilize locks to safely increment counters and set expiration times without race conditions.
- Fallback Mechanism: Implement a fallback in case central storage is unavailable to ensure the API remains operational.

🔑 **Follow-up**: What happens if Redis (or our central state store) fails? How do we ensure graceful degradation?

📂 [Link to Episode 2 Code](./ep2)


---


---



## How to Navigate the Series

Each episode is self-contained and focuses on solving a specific problem. The episodes are structured as follows:

1. **Scenario**: A real-world engineering problem you're likely to encounter.
2. **Gotcha**: The overlooked challenge that can arise when solving this problem.
3. **Solution**: The approach I’ve taken to solve the problem using Golang. This includes simplified code where appropriate and detailed comments in the codebase.
4. **Follow-up**: Additional considerations or edge cases that could be explored further.


---

## Contributing

I welcome contributions! If you have a suggestion for a scenario you'd like to see covered, feel free to [open an issue](https://github.com/blazingkevin/engineering-gotchas/issues) or submit a pull request with your proposed changes.

---
