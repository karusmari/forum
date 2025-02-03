document.addEventListener('DOMContentLoaded', function() {
    // Обработчик для кнопок реакций
    document.querySelectorAll('.like-btn, .dislike-btn').forEach(button => {
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