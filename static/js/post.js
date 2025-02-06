function sharePost(title) {
    if (navigator.share) {
        navigator.share({
            title: title,
            url: window.location.href
        })
        .catch(console.error);
    } else {
        // Fallback - copy URL to clipboard
        navigator.clipboard.writeText(window.location.href)
            .then(() => alert('Link copied to clipboard!'))
            .catch(console.error);
    }
} 