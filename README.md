# Reddit-Migrate

A simple interface to migrate Reddit account data from an old account to a new account using Reddit APIs.

![Home](./assets/app_home.png)

> **Caution** :warning: : This tool requires cookies from both the old and new Reddit accounts, which contain sensitive data (credentials). Never share these cookies with anyone.

:warning: This tool uses Reddit's public API via official OAuth. It is intended for personal use only. You must use your own Reddit API credentials. Use responsibly and within Reddit's [API Terms](https://www.reddit.com/dev/api/) and [Content Policy](https://redditinc.com/policies).

## Demo

Check out the demo on YouTube: [Watch Demo](https://youtu.be/cpwPjjkW2O4)

## Installation

The easiest way to run Reddit-Migrate is by downloading the latest pre-compiled binary for your operating system from the [**GitHub Releases page**](https://github.com/nileshnk/reddit-migrate/releases).

### Running a Release Binary

1.  Go to the [Releases page](https://github.com/nileshnk/reddit-migrate/releases).
2.  Download the appropriate ZIP archive for your operating system and architecture (e.g., `reddit-migrate-x.y.z-linux-amd64.zip`).

    - Windows: `reddit-migrate-x.y.z-windows-amd64.zip`, `reddit-migrate-x.y.z-windows-arm64.zip`
    - Linux: `reddit-migrate-x.y.z-linux-amd64.zip`, `reddit-migrate-x.y.z-linux-arm64.zip`
    - macOS (Intel & Apple Silicon): `reddit-migrate-x.y.z-macos-amd64.zip`, `reddit-migrate-x.y.z-macos-arm64.zip`
    - FreeBSD: `reddit-migrate-x.y.z-freebsd-amd64.zip`, `reddit-migrate-x.y.z-freebsd-arm64.zip`

    (where `x.y.z` is the version number)

3.  Extract the ZIP archive. This will create a folder (e.g., `reddit-migrate/`) containing the application executable (e.g., `reddit-migrate`, `reddit-migrate.exe`, or `reddit-migrate.app`) and a `public` folder with necessary UI assets.

4.  Make the binary executable (for Linux/macOS/FreeBSD if needed, though it should already have execute permissions from the build process):

    ```bash
    # Navigate into the extracted folder first
    cd reddit-migrate
    chmod +x ./reddit-migrate # Or ./reddit-migrate.app/Contents/MacOS/reddit-migrate for the .app bundle executable
    ```

    **For macOS users**: If you get a security warning or "unidentified developer" message when opening the `.app` bundle:

    1. Right-click (or Control-click) the `reddit-migrate.app` bundle in Finder.
    2. Select "Open" from the context menu.
    3. Click "Open" in the security dialog that appears.
    4. The app will now be saved as an exception to your security settings.

5.  Run the application from within the extracted folder:

    ```bash
    # For Linux/FreeBSD:
    ./reddit-migrate

    # For macOS .app bundles, you can usually double-click the .app bundle in Finder.
    # Alternatively, to run the raw macOS binary from terminal (ensure you are in the extracted 'reddit-migrate' directory):
    # ./bin/reddit-migrate
    # Or to run the .app bundle from terminal:
    # ./reddit-migrate.app/Contents/MacOS/reddit-migrate

    # For Windows:
    # Double-click the reddit-migrate.exe file or run from command prompt:
    .\reddit-migrate.exe
    ```

    By default, the application will start on `http://localhost:5005`. You can specify a custom address using the `--addr` flag:

    ```bash
    ./reddit-migrate --addr=":3000" # For Linux/FreeBSD
    ./bin/reddit-migrate --addr=":3000" # For macOS raw binary
    # Or for Windows:
    .\reddit-migrate.exe --addr=":3000"
    ```

### Building from Source

If you prefer to build the application from source:

1. Make sure you have Go installed on your system. If not, you can install it [here](https://go.dev/dl/).

2. Clone this repository and navigate to the project directory:

   ```bash
   git clone https://github.com/nileshnk/reddit-migrate.git
   cd reddit-migrate
   ```

3. Install the necessary Go dependencies:

   ```bash
   go mod tidy
   ```

4. Run the application:

   ```bash
   go run .
   ```

   Or build a binary:

   ```bash
   go build -o reddit-migrate
   ./reddit-migrate
   ```

5. You can also specify a custom address using the `--addr` flag when running:

   ```bash
   go run . --addr=":3000"
   # or if built:
   ./reddit-migrate --addr=":3000"
   ```

### Using Docker (for development or specific deployments)

While pre-compiled releases are recommended for most users, you can still use Docker.

1. Make sure Docker is installed and running on your system.

2. Clone this repository and navigate to the project directory (if you haven't already):

   ```bash
   git clone https://github.com/nileshnk/reddit-migrate.git
   cd reddit-migrate
   ```

3. Build the Docker image:

   ```bash
   docker build -t reddit-migrate-img .
   ```

4. Run the Docker container:

   ```bash
   docker run -d -p 5005:5005 --name reddit-migrate reddit-migrate-img
   ```

After setup, open the application in your browser at [http://localhost:5005](http://localhost:5005) or the custom address you provided during setup. Follow the steps below.

## Usage

> Migrating more than 50 saved posts may take additional time, with approximately 10 extra minutes required for every additional 50 posts. For example, migrating 100 posts could take around 10-15 minutes. Please leave the tab open without refreshing until you get a response. This is because of Reddit's Rate Limiting

### Steps:

1. **Retrieve Cookies**: Obtain the cookies for both your old and new Reddit accounts. You can follow this [video guide](./assets/cookie-retrieval.gif) for assistance.

   - Log in to Reddit in a desktop web browser.
   - Open a new tab and access the Network tab through developer tools (right-click and select Inspect).
   - Visit the URL [https://www.reddit.com/api/me.json](https://www.reddit.com/api/me.json) or any other Reddit page.
   - Locate the new request that appeared in the network tab.
   - Find and select the whole cookie (you can use triple-click to select).
   - Copy the cookie.
   - Repeat the same process for the other Reddit account.

2. **Paste Cookies**: Paste both cookies accordingly and verify their correctness.

3. **Select Options**: Choose the migration options based on your requirements.

4. **Submit**: Click the submit button to initiate the migration.

## Code Description

This tool uses Reddit's APIs to interact with Reddit servers using cookies from respective user accounts. Here's a brief overview:

- **Verification**: Clicking the verify button verifies the cookies. It checks for the presence of an access token in the cookie and validates the response from Reddit.

For subreddit migrations, the following steps are performed:

- Request a list of all subscribed subreddits and store their display names.
- Pass the names to a function that subscribes to the specified subreddits.
- Process subreddits in chunks of 100 names per request.
- Handle user accounts that are subscribed separately.
- Return success and failure counts.

For saved-posts migrations, these steps are followed:

- Request a list of all saved posts and store their full names.
- Pass the full names to a function that saves the posts.
- The function makes one API request per saved post, which may be subject to rate limits.
- Return success and failure counts.

Initially, this tool was written in JavaScript but was later rewritten in Go as an exercise to become familiar with the language.

## Contact

For any queries or suggestions, please feel free to contact me at [mail@inilesh.com](mailto:mail@inilesh.com).
