#!/usr/bin/env node

const https = require("https");
const http = require("http");
const { URL } = require("url");

// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2);
  const config = {
    subreddits: null, // Changed from 0 to null to distinguish between explicitly set 0 and unset
    posts: null, // Changed from 0 to null to distinguish between explicitly set 0 and unset
    cookie: null,
    unfollowAll: false,
    unsaveAll: false,
  };

  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case "--subreddits":
        config.subreddits = parseInt(args[i + 1]) || 0;
        i++;
        break;
      case "--posts":
        config.posts = parseInt(args[i + 1]) || 0;
        i++;
        break;
      case "--cookie":
        config.cookie = args[i + 1];
        i++;
        break;
      case "--unfollowAll":
        config.unfollowAll = true;
        break;
      case "--unsaveAll":
        config.unsaveAll = true;
        break;
      case "--help":
      case "-h":
        console.log(`
Reddit Test Data Generator

Usage: node reddit-test-data.js [options]

Options:
  --subreddits <number>   Number of subreddits to follow (default: 10)
  --posts <number>        Number of posts to save (default: 10)
  --cookie <cookie>       Reddit cookie string
  --unfollowAll          Unfollow all currently followed subreddits
  --unsaveAll            Unsave all currently saved posts
  --help, -h             Show this help message

Examples:
  node reddit-test-data.js --subreddits 50 --posts 100
  node reddit-test-data.js --subreddits 25 --posts 50 --cookie "reddit_session=..."
  node reddit-test-data.js --unfollowAll --cookie "reddit_session=..."
  node reddit-test-data.js --unsaveAll --cookie "reddit_session=..."
                `);
        process.exit(0);
        break;
    }
  }

  // Set defaults only if not doing cleanup operations and values weren't explicitly set
  if (!config.unfollowAll && !config.unsaveAll) {
    if (config.subreddits === null) config.subreddits = 10;
    if (config.posts === null) config.posts = 10;
  } else {
    // For cleanup operations, ensure we don't accidentally trigger follow/save operations
    if (config.subreddits === null) config.subreddits = 0;
    if (config.posts === null) config.posts = 0;
  }

  return config;
}

// Reddit cookie - you can paste your cookie here or use --cookie argument
const DEFAULT_COOKIE = ``.replace(/\s+/g, "").replace(/;$/, "");

function extractTokenFromCookie(cookie) {
  if (!cookie) return null;
  const match = cookie.match(/token_v2=([^;]+)/);
  return match ? match[1] : null;
}

class RedditAPI {
  constructor(cookie) {
    this.cookie = cookie || DEFAULT_COOKIE;
    this.baseURL = "https://oauth.reddit.com";
    const token = extractTokenFromCookie(this.cookie);
    this.headers = {
      "User-Agent": "RedditMigrationTestScript/1.0",
      Accept: "application/json",
      "Content-Type": "application/x-www-form-urlencoded",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    };
    this.delay = 1000; // 1 second delay between requests
  }

  async makeRequest(endpoint, method = "GET", data = null) {
    return new Promise((resolve, reject) => {
      const url = new URL(endpoint, this.baseURL);
      const options = {
        method,
        headers: this.headers,
        timeout: 10000,
      };

      const req = https.request(url, options, (res) => {
        let body = "";
        res.on("data", (chunk) => (body += chunk));
        res.on("end", () => {
          try {
            if (res.statusCode >= 200 && res.statusCode < 300) {
              resolve(JSON.parse(body));
            } else {
              reject(new Error(`HTTP ${res.statusCode}: ${body}`));
            }
          } catch (e) {
            reject(new Error(`Invalid JSON response: ${body}`));
          }
        });
      });

      req.on("error", reject);
      req.on("timeout", () => {
        req.destroy();
        reject(new Error("Request timeout"));
      });

      if (data && method !== "GET") {
        req.write(typeof data === "string" ? data : JSON.stringify(data));
      }

      req.end();
    });
  }

  async delayTime(ms = this.delay) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async getPopularSubreddits(totalLimit = 100) {
    console.log(`🔍 Fetching ${totalLimit} popular subreddits...`);
    const allSubreddits = [];
    const batchSize = 100; // Reddit API limit per request
    let after = null;

    try {
      while (allSubreddits.length < totalLimit) {
        const remaining = totalLimit - allSubreddits.length;
        const limit = Math.min(batchSize, remaining);

        let endpoint = `/subreddits/popular?limit=${limit}`;
        if (after) {
          endpoint += `&after=${after}`;
        }

        console.log(
          `   Fetching batch: ${allSubreddits.length + 1}-${
            allSubreddits.length + limit
          }`
        );

        const response = await this.makeRequest(endpoint);
        const subreddits = response.data.children
          .map((child) => child.data)
          .filter((subreddit) => !subreddit.over18 && !subreddit.quarantine);

        if (subreddits.length === 0) {
          console.log("   No more subreddits available");
          break;
        }

        allSubreddits.push(...subreddits);
        after = response.data.after;

        // If no more pages available
        if (!after) {
          console.log("   Reached end of available subreddits");
          break;
        }

        // Small delay between requests
        if (allSubreddits.length < totalLimit) {
          await this.delayTime(500);
        }
      }

      console.log(`✅ Fetched ${allSubreddits.length} total subreddits`);
      return allSubreddits;
    } catch (error) {
      console.error("Error fetching popular subreddits:", error.message);
      return allSubreddits; // Return what we have so far
    }
  }

  async getRandomPosts(subreddit = "all", totalLimit = 100) {
    console.log(`🔍 Fetching ${totalLimit} posts from r/${subreddit}...`);
    const allPosts = [];
    const batchSize = 100; // Reddit API limit per request
    let after = null;

    try {
      while (allPosts.length < totalLimit) {
        const remaining = totalLimit - allPosts.length;
        const limit = Math.min(batchSize, remaining);

        let endpoint = `/r/${subreddit}/hot?limit=${limit}`;
        if (after) {
          endpoint += `&after=${after}`;
        }

        console.log(
          `   Fetching batch: ${allPosts.length + 1}-${allPosts.length + limit}`
        );

        const response = await this.makeRequest(endpoint);
        const posts = response.data.children
          .map((child) => child.data)
          .filter(
            (post) => !post.over18 && !post.spoiler && post.subreddit !== "all"
          );

        if (posts.length === 0) {
          console.log("   No more posts available");
          break;
        }

        allPosts.push(...posts);
        after = response.data.after;

        // If no more pages available
        if (!after) {
          console.log("   Reached end of available posts");
          break;
        }

        // Small delay between requests
        if (allPosts.length < totalLimit) {
          await this.delayTime(500);
        }
      }

      console.log(`✅ Fetched ${allPosts.length} total posts`);
      return allPosts;
    } catch (error) {
      console.error(`Error fetching posts from r/${subreddit}:`, error.message);
      return allPosts; // Return what we have so far
    }
  }

  async followSubreddit(subredditName) {
    try {
      const data = `sr_name=${subredditName}&action=sub`;
      await this.makeRequest("/api/subscribe", "POST", data);
      return true;
    } catch (error) {
      console.error(`Error following r/${subredditName}:`, error.message);
      return false;
    }
  }

  async savePost(postId) {
    try {
      const data = `id=t3_${postId}`;
      await this.makeRequest("/api/save", "POST", data);
      return true;
    } catch (error) {
      console.error(`Error saving post ${postId}:`, error.message);
      return false;
    }
  }

  async getCurrentUser() {
    try {
      const response = await this.makeRequest("/api/v1/me");
      return response;
    } catch (error) {
      console.error("Error getting current user:", error.message);
      return null;
    }
  }

  async unfollowSubreddit(subredditName) {
    try {
      const data = `sr_name=${subredditName}&action=unsub`;
      await this.makeRequest("/api/subscribe", "POST", data);
      return true;
    } catch (error) {
      console.error(`Error unfollowing r/${subredditName}:`, error.message);
      return false;
    }
  }

  async unsavePost(postId) {
    try {
      const data = `id=${postId}`;
      await this.makeRequest("/api/unsave", "POST", data);
      return true;
    } catch (error) {
      console.error(`Error unsaving post ${postId}:`, error.message);
      return false;
    }
  }

  async getFollowedSubreddits() {
    try {
      console.log("🔍 Fetching followed subreddits...");
      const response = await this.makeRequest(
        "/subreddits/mine/subscriber?limit=100"
      );
      return response.data.children.map((child) => child.data.display_name);
    } catch (error) {
      console.error("Error fetching followed subreddits:", error.message);
      return [];
    }
  }

  async getSavedPosts(limit = 100) {
    try {
      console.log("🔍 Fetching saved posts...");

      // First get the current user to get their username
      const user = await this.getCurrentUser();
      if (!user || !user.name) {
        throw new Error("Could not get current user");
      }

      const response = await this.makeRequest(
        `/user/${user.name}/saved?limit=${limit}`
      );
      return response.data.children.map((child) => ({
        id: child.data.name,
        title: child.data.title || child.data.body || "Comment",
        subreddit: child.data.subreddit,
      }));
    } catch (error) {
      console.error("Error fetching saved posts:", error.message);
      return [];
    }
  }

  async bulkSavePosts(postIds, chunkSize = 10) {
    if (!Array.isArray(postIds) || postIds.length === 0) {
      return { successCount: 0, failedCount: 0, failedPosts: [], error: false };
    }
    let successCount = 0;
    let failedCount = 0;
    let failedPosts = [];
    let error = false;
    for (let i = 0; i < postIds.length; i += chunkSize) {
      const chunk = postIds.slice(i, i + chunkSize);
      const chunkResults = await Promise.all(
        chunk.map(async (postId) => {
          try {
            const result = await this.savePost(postId);
            return { postId, success: result };
          } catch (e) {
            return { postId, success: false };
          }
        })
      );
      chunkResults.forEach(({ postId, success }) => {
        if (success) {
          successCount++;
        } else {
          failedCount++;
          failedPosts.push(postId);
          error = true;
        }
      });
      // Delay between chunks to avoid rate limiting
      if (i + chunkSize < postIds.length) {
        await this.delayTime();
      }
    }
    return { successCount, failedCount, failedPosts, error };
  }

  async followSubredditChunk(subredditNames) {
    if (!Array.isArray(subredditNames) || subredditNames.length === 0) {
      return {
        successCount: 0,
        failedCount: 0,
        failedSubreddits: [],
        error: false,
      };
    }

    // Reddit API can handle multiple subreddits in one request, but let's be safe with smaller chunks
    const chunkSize = 25; // Conservative chunk size to avoid issues
    let totalSuccess = 0;
    let totalFailed = 0;
    let allFailedSubreddits = [];

    for (let i = 0; i < subredditNames.length; i += chunkSize) {
      const chunk = subredditNames.slice(i, i + chunkSize);
      const sr_name = chunk.join(",");
      const data = `sr_name=${encodeURIComponent(
        sr_name
      )}&action=sub&api_type=json`;

      try {
        await this.makeRequest("/api/subscribe", "POST", data);
        totalSuccess += chunk.length;
        console.log(`✅ Successfully followed: ${chunk.join(", ")}`);
      } catch (error) {
        console.error(
          `❌ Failed to follow chunk: ${chunk.join(", ")} - ${error.message}`
        );
        totalFailed += chunk.length;
        allFailedSubreddits.push(...chunk);
      }

      // Delay between chunks
      if (i + chunkSize < subredditNames.length) {
        await this.delayTime();
      }
    }

    return {
      successCount: totalSuccess,
      failedCount: totalFailed,
      failedSubreddits: allFailedSubreddits,
      error: totalFailed > 0,
    };
  }
}

class TestDataGenerator {
  constructor(config) {
    this.config = config;
    this.reddit = new RedditAPI(config.cookie);
    this.stats = {
      subredditsFollowed: 0,
      postsSaved: 0,
      subredditsUnfollowed: 0,
      postsUnsaved: 0,
      errors: 0,
    };
  }

  async validateAuthentication() {
    console.log("🔐 Validating authentication...");
    const user = await this.reddit.getCurrentUser();
    if (user && user.name) {
      console.log(`✅ Authenticated as: ${user.name}`);
      return true;
    } else {
      console.error("❌ Authentication failed. Please check your cookie.");
      return false;
    }
  }

  // Fisher-Yates shuffle algorithm for better randomization
  shuffleArray(array) {
    const shuffled = [...array];
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    return shuffled;
  }

  async followRandomSubreddits() {
    console.log(
      `\n🎯 Following ${this.config.subreddits} random subreddits...`
    );

    // Fetch more subreddits than needed to ensure good randomization
    const fetchCount = Math.max(this.config.subreddits * 3, 300);
    const subreddits = await this.reddit.getPopularSubreddits(fetchCount);

    if (subreddits.length === 0) {
      console.error("❌ No subreddits found");
      return;
    }

    console.log(`📊 Fetched ${subreddits.length} subreddits for randomization`);

    // Properly shuffle and take the requested amount
    const shuffled = this.shuffleArray(subreddits);
    const selected = shuffled.slice(0, this.config.subreddits);

    console.log(`📋 Selected ${selected.length} subreddits to follow:`);
    selected.forEach((sub, index) => {
      console.log(
        `   ${index + 1}. r/${
          sub.display_name
        } (${sub.subscribers?.toLocaleString()} subscribers)`
      );
    });

    // Follow subreddits using the improved chunking method
    const subredditNames = selected.map((sub) => sub.display_name);
    const result = await this.reddit.followSubredditChunk(subredditNames);

    this.stats.subredditsFollowed = result.successCount;
    this.stats.errors += result.failedCount;

    if (result.failedSubreddits.length > 0) {
      console.log(`❌ Failed to follow: ${result.failedSubreddits.join(", ")}`);
    }

    console.log(
      `✅ Successfully followed: ${result.successCount}/${this.config.subreddits} subreddits`
    );
  }

  async saveRandomPosts() {
    console.log(`\n💾 Saving ${this.config.posts} random posts...`);

    // Fetch more posts than needed to ensure good randomization
    const fetchCount = Math.max(this.config.posts * 2, 200);
    const posts = await this.reddit.getRandomPosts("all", fetchCount);

    if (posts.length === 0) {
      console.error("❌ No posts found");
      return;
    }

    console.log(`📊 Fetched ${posts.length} posts for randomization`);

    // Properly shuffle and take the requested amount
    const shuffled = this.shuffleArray(posts);
    const selected = shuffled.slice(0, this.config.posts);

    console.log(`📋 Selected ${selected.length} posts to save:`);
    selected.forEach((post, index) => {
      const title =
        post.title.length > 60
          ? post.title.substring(0, 60) + "..."
          : post.title;
      console.log(`   ${index + 1}. "${title}" from r/${post.subreddit}`);
    });

    for (let i = 0; i < selected.length; i++) {
      const post = selected[i];
      const shortTitle =
        post.title.length > 50
          ? post.title.substring(0, 50) + "..."
          : post.title;
      process.stdout.write(`Saving "${shortTitle}"... `);

      const success = await this.reddit.savePost(post.id);
      if (success) {
        this.stats.postsSaved++;
        console.log("✅");
      } else {
        this.stats.errors++;
        console.log("❌");
      }

      // Progress indicator
      const progress = Math.round(((i + 1) / selected.length) * 100);
      console.log(`Progress: ${progress}% (${i + 1}/${selected.length})`);

      await this.reddit.delayTime();
    }
  }

  async unfollowAllSubreddits() {
    console.log("\n🎯 Unfollowing all subreddits...");

    const subreddits = await this.reddit.getFollowedSubreddits();
    if (subreddits.length === 0) {
      console.log("✅ No subreddits to unfollow");
      return;
    }

    console.log(`📋 Found ${subreddits.length} subreddits to unfollow:`);
    subreddits.forEach((sub, index) => {
      console.log(`   ${index + 1}. r/${sub}`);
    });

    for (let i = 0; i < subreddits.length; i++) {
      const subreddit = subreddits[i];
      process.stdout.write(`Unfollowing r/${subreddit}... `);

      const success = await this.reddit.unfollowSubreddit(subreddit);
      if (success) {
        this.stats.subredditsUnfollowed++;
        console.log("✅");
      } else {
        this.stats.errors++;
        console.log("❌");
      }

      // Progress indicator
      const progress = Math.round(((i + 1) / subreddits.length) * 100);
      console.log(`Progress: ${progress}% (${i + 1}/${subreddits.length})`);

      await this.reddit.delayTime();
    }
  }

  async unsaveAllPosts() {
    console.log("\n💾 Unsaving all posts...");

    const posts = await this.reddit.getSavedPosts();
    if (posts.length === 0) {
      console.log("✅ No posts to unsave");
      return;
    }

    console.log(`📋 Found ${posts.length} posts to unsave:`);
    posts.forEach((post, index) => {
      const title =
        post.title.length > 60
          ? post.title.substring(0, 60) + "..."
          : post.title;
      console.log(
        `   ${index + 1}. "${title}" from r/${post.subreddit || "unknown"}`
      );
    });

    for (let i = 0; i < posts.length; i++) {
      const post = posts[i];
      const shortTitle =
        post.title.length > 50
          ? post.title.substring(0, 50) + "..."
          : post.title;
      process.stdout.write(`Unsaving "${shortTitle}"... `);

      const success = await this.reddit.unsavePost(post.id);
      if (success) {
        this.stats.postsUnsaved++;
        console.log("✅");
      } else {
        this.stats.errors++;
        console.log("❌");
      }

      // Progress indicator
      const progress = Math.round(((i + 1) / posts.length) * 100);
      console.log(`Progress: ${progress}% (${i + 1}/${posts.length})`);

      await this.reddit.delayTime();
    }
  }

  async run() {
    console.log("🚀 Reddit Test Data Generator");
    console.log("===============================");

    // Show what operations will be performed
    const operations = [];
    if (this.config.unfollowAll) operations.push("unfollow all subreddits");
    if (this.config.unsaveAll) operations.push("unsave all posts");
    if (!this.config.unfollowAll && !this.config.unsaveAll) {
      if (this.config.subreddits > 0)
        operations.push(`follow ${this.config.subreddits} subreddits`);
      if (this.config.posts > 0)
        operations.push(`save ${this.config.posts} posts`);
    }

    console.log(`Operations: ${operations.join(", ")}`);

    if (!(await this.validateAuthentication())) {
      process.exit(1);
    }

    const startTime = Date.now();

    // Handle cleanup operations first (and exclusively)
    if (this.config.unfollowAll) {
      await this.unfollowAllSubreddits();
    }

    if (this.config.unsaveAll) {
      await this.unsaveAllPosts();
    }

    // Only do normal operations if not doing cleanup
    if (!this.config.unfollowAll && !this.config.unsaveAll) {
      if (this.config.subreddits > 0) {
        await this.followRandomSubreddits();
      }

      if (this.config.posts > 0) {
        await this.saveRandomPosts();
      }
    }

    const endTime = Date.now();
    const duration = Math.round((endTime - startTime) / 1000);

    console.log("\n📊 Summary:");
    console.log("===========");
    if (this.stats.subredditsFollowed > 0) {
      console.log(
        `✅ Subreddits followed: ${this.stats.subredditsFollowed}/${this.config.subreddits}`
      );
    }
    if (this.stats.postsSaved > 0) {
      console.log(
        `✅ Posts saved: ${this.stats.postsSaved}/${this.config.posts}`
      );
    }
    if (this.stats.subredditsUnfollowed > 0) {
      console.log(
        `✅ Subreddits unfollowed: ${this.stats.subredditsUnfollowed}`
      );
    }
    if (this.stats.postsUnsaved > 0) {
      console.log(`✅ Posts unsaved: ${this.stats.postsUnsaved}`);
    }
    console.log(`❌ Errors: ${this.stats.errors}`);
    console.log(`⏱️  Duration: ${duration} seconds`);

    if (this.stats.errors > 0) {
      console.log("\n⚠️  Some operations failed. This could be due to:");
      console.log("   - Rate limiting");
      console.log("   - Invalid cookie/authentication");
      console.log("   - Network issues");
      console.log("   - Already subscribed/saved items");
    }

    console.log("\n🎉 Operation complete!");
    if (this.config.unfollowAll || this.config.unsaveAll) {
      console.log("Cleanup operations finished.");
    } else {
      console.log("You can now test your migration script with this data.");
    }
  }
}

// Main execution
async function main() {
  const config = parseArgs();

  if (!config.cookie && DEFAULT_COOKIE.includes("your_session_here")) {
    console.error("❌ No cookie provided!");
    console.log("\nPlease either:");
    console.log("1. Edit the DEFAULT_COOKIE in this script, or");
    console.log("2. Use --cookie argument with your Reddit cookie");
    console.log(
      '\nExample: node reddit-test-data.js --cookie "reddit_session=abc123;token_v2=xyz789"'
    );
    process.exit(1);
  }

  // Updated validation logic - more explicit about when operations are needed
  const hasCleanupOperation = config.unfollowAll || config.unsaveAll;
  const hasNormalOperation =
    (config.subreddits !== null && config.subreddits > 0) ||
    (config.posts !== null && config.posts > 0);

  if (!hasCleanupOperation && !hasNormalOperation) {
    console.error(
      "❌ Please specify at least one operation: --subreddits, --posts, --unfollowAll, or --unsaveAll"
    );
    process.exit(1);
  }

  const generator = new TestDataGenerator(config);
  await generator.run();
}

// Handle graceful shutdown
process.on("SIGINT", () => {
  console.log("\n\n🛑 Interrupted by user", new Date().toISOString());
  process.exit(0);
});

process.on("unhandledRejection", (error) => {
  console.error("❌ Unhandled error:", error.message);
  process.exit(1);
});

if (require.main === module) {
  main().catch((error) => {
    console.error("❌ Fatal error:", error.message);
    process.exit(1);
  });
}

module.exports = { TestDataGenerator, RedditAPI };
