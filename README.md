# Go Download Mangaer - [Repositiry Link](https://github.com/mjghr/tech-download-manager)
A file downloader that leverages Go concurrency to download public files off the Internet

## Features:
+ **Concurrent** Downloads: Split files into chunks and download them concurrently for faster downloads.
+ **Pause/Resume**: Pause and resume downloads at any time.
+ **Speed Limit**: Set download speed limits to control bandwidth usage.
+ **Queue Management**: Manage multiple downloads in a queue with configurable concurrency limits.
+ **Progress Tracking**: Track download progress in real-time.
+ **Time Window**: Schedule downloads to run within a specific time window.
+ **Temporary Files**: Use temporary files for resumable downloads and cleanup after completion.
+ **Error Handling**: Retry failed chunks and handle errors gracefully.

## Implementation:
We might be tempted to GET the entire file at once, but if it is a large file, it can cause certain limitations, which are mitigated if we Divide and Conquer i.e, we break down the large file into certain C chunks and GET each of the C chunks individually. Also, it has other merits.

## How to Run:

Follow these steps to set up and alos before running the application, Ensure you’re using Go 1.20 or higher

### Step 1: Clone the Repository

Clone the TDM repository to your local machine:

```bash
git clone https://github.com/mjghr/tech-download-manager.git
cd tech-download-manager
```

### Step 2: Run the code

run the programm using this command

```bash
go run cmd/main.go
```
### Step 3: 
use the app as you see fit

## Contributors:
+ Nima Alighardashi - 401100466
+ Mohammad Javad Gharegozlou - 401170134 - mjghrfr@gmail.com 
+ Shaygan Adim - 401109971

<div style="display: flex; gap: 25px;">
  <img src="https://iili.io/3zG949f.png" alt="Description of the image" width="250" height=300 style="border-radius: 55px;" />
  <img src="https://iili.io/3zG96u4.png" alt="Description of the image" width="250" height=300 style="border-radius: 55px;" />
  <img src="https://iili.io/3zG9Pwl.png" alt="Description of the image" width="250" height=300 style="border-radius: 55px;" />
</div>

