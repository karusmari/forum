document.addEventListener('DOMContentLoaded', function() {
    //finding all the buttons with the class reactions
    document.querySelectorAll('.post .reactions button').forEach(button => {
        button.addEventListener('click', async function(e) {
            e.preventDefault();
            
            const postId = this.dataset.postId;
            const type = this.dataset.type;
            
            console.log('Clicking post reaction button:', {
                postId: this.dataset.postId,
                type: this.dataset.type
            });
            
            try {
                const response = await fetch('/api/react', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        post_id: parseInt(postId),
                        type: type
                    })
                });

                console.log('Sending request:', {
                    url: '/api/react',
                    body: {
                        post_id: parseInt(postId),
                        type: type
                    }
                });

                if (!response.ok) {
                    throw new Error('Network response was not ok');
                }

                const data = await response.json();
                if (data.success) {
                    // Update the counters
                    const post = this.closest('.post');
                    post.querySelector('.like-btn .likes-count').textContent = data.likes;
                    post.querySelector('.dislike-btn .dislikes-count').textContent = data.dislikes;
                    
                    // Update the active state of the buttons
                    if (this.classList.contains('active')) {
                        this.classList.remove('active');
                    } else {
                        post.querySelectorAll('.reactions button').forEach(btn => {
                            btn.classList.remove('active');
                        });
                        this.classList.add('active');
                    }
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error updating reaction');
            }
        });
    });

    // Handler for comment reactions
    document.querySelectorAll('.comment .reactions button').forEach(button => {
        button.addEventListener('click', async function(e) {
            e.preventDefault();
            
            const commentId = this.dataset.commentId;
            const type = this.dataset.type;
            
            try {
                const response = await fetch('/api/comment/react', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        comment_id: parseInt(commentId),
                        type: type
                    })
                });

                if (!response.ok) {
                    throw new Error('Network response was not ok');
                }

                const data = await response.json();
                if (data.success) {
                    // Update the counters
                    const comment = this.closest('.comment');
                    comment.querySelector('.like-btn .likes-count').textContent = data.likes;
                    comment.querySelector('.dislike-btn .dislikes-count').textContent = data.dislikes;
                    
                    // Update the active state of the buttons
                    if (this.classList.contains('active')) {
                        this.classList.remove('active');
                    } else {
                        comment.querySelectorAll('.reactions button').forEach(btn => {
                            btn.classList.remove('active');
                        });
                        this.classList.add('active');
                    }
                }
            } catch (error) {
                console.error('Error:', error);
                alert('Error updating reaction');
            }
        });
    });
}); 