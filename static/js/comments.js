function toggleEditComment(commentId) {
    const content = document.getElementById(`comment-content-${commentId}`);
    const form = document.getElementById(`edit-form-${commentId}`);
    
    if (content.style.display !== 'none') {
        content.style.display = 'none';
        form.style.display = 'block';
    } else {
        content.style.display = 'block';
        form.style.display = 'none';
    }
}

// Обработка реакций на комментарии
document.addEventListener('click', function(e) {
    if (e.target.matches('[data-comment-id]')) {
        const commentId = e.target.dataset.commentId;
        const type = e.target.dataset.type;
        
        fetch('/api/comment/react', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                comment_id: commentId,
                type: type
            })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const likesCount = e.target.querySelector('.likes-count');
                const dislikesCount = e.target.querySelector('.dislikes-count');
                if (likesCount) likesCount.textContent = data.likes;
                if (dislikesCount) dislikesCount.textContent = data.dislikes;
                
                // Обновляем классы активности
                if (type === 'like') {
                    e.target.classList.toggle('active');
                    e.target.nextElementSibling?.classList.remove('active');
                } else {
                    e.target.classList.toggle('active');
                    e.target.previousElementSibling?.classList.remove('active');
                }
            }
        })
        .catch(error => console.error('Error:', error));
    }
}); 