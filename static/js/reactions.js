document.addEventListener('DOMContentLoaded', function() {
    // Обработчик только для кнопок реакций постов
    document.querySelectorAll('.like-btn:not(.comment-like-btn), .dislike-btn:not(.comment-dislike-btn)').forEach(button => {
        button.addEventListener('click', async function(e) {
            e.preventDefault();

            const postId = this.dataset.postId;
            const type = this.dataset.type;

            try {
                const response = await fetch('/api/react', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: `post_id=${postId}&type=${type}`
                });

                if (response.ok) {
                    const data = await response.json();
                    
                    // Обновляем количество лайков и дислайков
                    const post = this.closest('article');
                    post.querySelector('.like-btn').textContent = `👍 ${data.likes}`;
                    post.querySelector('.dislike-btn').textContent = `👎 ${data.dislikes}`;

                    // Обновляем активное состояние кнопок
                    post.querySelector('.like-btn').classList.toggle('active', type === 'like');
                    post.querySelector('.dislike-btn').classList.toggle('active', type === 'dislike');
                } else {
                    if (response.status === 401) {
                        window.location.href = '/login';
                    } else {
                        alert('Error saving reaction');
                    }
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error saving reaction');
            }
        });
    });
});

// Для реакций на посты
function reactToPost(postId, type) {
    fetch('/api/react', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: `post_id=${postId}&type=${type}`
    })
    .then(response => response.json())
    .then(data => {
        document.querySelector(`#post-${postId} .likes-count`).textContent = data.likes;
        document.querySelector(`#post-${postId} .dislikes-count`).textContent = data.dislikes;
    })
    .catch(error => console.error('Error:', error));
}

// Добавляем новую функцию для реакций на комментарии
function reactToComment(commentId, type) {
    fetch('/api/comment/react', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: `comment_id=${commentId}&type=${type}`
    })
    .then(response => response.json())
    .then(data => {
        document.querySelector(`#comment-${commentId} .likes-count`).textContent = data.likes;
        document.querySelector(`#comment-${commentId} .dislikes-count`).textContent = data.dislikes;
    })
    .catch(error => console.error('Error:', error));
} 