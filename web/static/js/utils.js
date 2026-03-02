// Pure utility functions — no state dependencies

export function formatNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + "M";
  if (num >= 1000) return (num / 1000).toFixed(1) + "K";
  return num.toString();
}

export function formatTimeAgo(timestamp) {
  const now = Date.now() / 1000;
  const diff = now - timestamp;

  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 2592000) return `${Math.floor(diff / 86400)}d ago`;
  return `${Math.floor(diff / 2592000)}mo ago`;
}

export function getPostImageUrl(post) {
  const imageData = post.image_data;
  if (imageData.preview_url) return imageData.preview_url;
  if (imageData.thumbnail_url) return imageData.thumbnail_url;
  if (imageData.high_res_url) return imageData.high_res_url;
  return null;
}

export function getMediaTypeIcon(mediaType) {
  switch (mediaType) {
    case "image":
      return "\uD83D\uDDBC\uFE0F";
    case "video":
      return "\uD83C\uDFA5";
    case "gallery":
      return "\uD83D\uDDBC\uFE0F";
    case "link":
      return "\uD83D\uDD17";
    default:
      return "\uD83D\uDCC4";
  }
}
