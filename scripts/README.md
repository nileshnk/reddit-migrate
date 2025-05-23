# Reddit Migration Test Scripts

This directory contains utility scripts for testing the Reddit migration functionality.

## Reddit Test Data Generator

The `reddit-test-data.js` script allows you to automatically follow random non-NSFW subreddits and save random non-NSFW posts to generate test data for your Reddit migration workflows.

### Prerequisites

- Node.js (version 12 or higher)
- A valid Reddit cookie from your browser

### Getting Your Reddit Cookie

1. Open your browser and log into Reddit
2. Open Developer Tools (F12)
3. Go to Application/Storage → Cookies → reddit.com
4. Copy the cookie string (all values separated by semicolons)

### Usage

#### Basic Usage

```bash
# Follow 10 subreddits and save 10 posts (defaults)
node scripts/reddit-test-data.js

# Custom amounts
node scripts/reddit-test-data.js --subreddits 50 --posts 100

# Only follow subreddits (no posts)
node scripts/reddit-test-data.js --subreddits 25 --posts 0

# Only save posts (no subreddits)
node scripts/reddit-test-data.js --subreddits 0 --posts 50
```

#### Using Cookie Argument

```bash
node scripts/reddit-test-data.js --subreddits 100 --posts 100 --cookie "reddit_session=abc123;token_v2=xyz789"
```

#### Or Edit the Script

You can paste your cookie directly in the script by editing the `DEFAULT_COOKIE` constant:

```javascript
const DEFAULT_COOKIE = `
reddit_session=your_actual_session_here;
token_v2=your_actual_token_here;
csv=your_actual_csv_here;
session_tracker=your_actual_tracker_here;
`
  .replace(/\s+/g, "")
  .replace(/;$/, "");
```

### Command Line Options

| Option                  | Description                    | Default             |
| ----------------------- | ------------------------------ | ------------------- |
| `--subreddits <number>` | Number of subreddits to follow | 10                  |
| `--posts <number>`      | Number of posts to save        | 10                  |
| `--cookie <string>`     | Reddit cookie string           | Uses DEFAULT_COOKIE |
| `--help`, `-h`          | Show help message              | -                   |

### Features

- ✅ **Safe**: Only processes non-NSFW content
- ✅ **Random**: Selects random subreddits and posts for variety
- ✅ **Rate-limited**: Includes delays to avoid hitting Reddit's API limits
- ✅ **Progress tracking**: Shows real-time progress and statistics
- ✅ **Error handling**: Graceful error handling and reporting
- ✅ **Authentication check**: Validates your cookie before starting

### Example Output

```bash
🚀 Reddit Test Data Generator
===============================
Target: 25 subreddits, 50 posts

🔐 Validating authentication...
✅ Authenticated as: your_username

🎯 Following 25 random subreddits...
📋 Selected 25 subreddits to follow:
   1. r/programming (2,500,000 subscribers)
   2. r/science (25,000,000 subscribers)
   ...

Following r/programming... ✅
Progress: 4% (1/25)
...

💾 Saving 50 random posts...
📋 Selected 50 posts to save:
   1. "Amazing discovery in quantum computing" from r/science
   2. "New JavaScript framework released" from r/programming
   ...

📊 Summary:
===========
✅ Subreddits followed: 25/25
✅ Posts saved: 50/50
❌ Errors: 0
⏱️ Duration: 180 seconds

🎉 Test data generation complete!
You can now test your migration script with this data.
```

### Testing Your Migration

After running this script, you'll have:

- Random subreddits in your "old" account's subscriptions
- Random saved posts in your "old" account

You can then test your migration tool by:

1. Using this account as the "old" account
2. Creating or using another account as the "new" account
3. Running your migration to transfer the generated data

### Troubleshooting

#### Authentication Issues

- Make sure your cookie is complete and current
- Try logging out and back into Reddit to get a fresh cookie
- Check that all required cookie components are included

#### Rate Limiting

- The script includes automatic delays between requests
- If you hit rate limits, try reducing the numbers or running later
- Reddit has stricter limits during peak hours

#### Network Issues

- Ensure you have a stable internet connection
- Some corporate networks may block Reddit API calls
- Try running from a different network if issues persist

### Safety Notes

- This script only reads from and writes to Reddit APIs
- It never deletes or modifies existing data
- All operations are reversible through Reddit's interface
- Only non-NSFW content is processed
- Respects Reddit's rate limiting guidelines

### Integration with Migration Tool

This script generates the perfect test data for your migration tool because:

- It creates real Reddit subscriptions and saved posts
- The data structure matches exactly what your migration tool expects
- You can control the exact amount of test data
- All content is safe and non-controversial
