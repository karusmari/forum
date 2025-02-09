function filterPosts(filter) {
    const posts = document.querySelectorAll('.post-preview');
    
    posts.forEach(post => {
        let show = false;
        switch(filter) {
            case 'my':
                show = post.dataset.isMine === "true";
                break;
            case 'liked':
                show = post.dataset.isLiked === "true";
                break;
            default: // 'all'
                show = true;
                break;
        }
        
        post.style.display = show ? 'block' : 'none';
        if (show) {
            post.style.opacity = '1';
        } else {
            post.style.opacity = '0';
        }
    });

    // update active button
    document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    event.target.classList.add('active');
}