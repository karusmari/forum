//this function enables the user to edit the comment
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

//adding the click event listener to the document
document.addEventListener('click', function(e) {
    if (e.target.matches('[data-comment-id]')) {
        const commentId = e.target.dataset.commentId;
        const type = e.target.dataset.type; //either like or dislike
        
        //creating a post request to the server
        fetch('/api/comment/react', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            //sending the comment id and the type of reaction to the server
            body: JSON.stringify({
                comment_id: commentId,
                type: type
            })
        })
        //if server responds with a success message, update the likes and dislikes count
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const likesCount = e.target.querySelector('.likes-count');
                const dislikesCount = e.target.querySelector('.dislikes-count');
                if (likesCount) likesCount.textContent = data.likes;
                if (dislikesCount) dislikesCount.textContent = data.dislikes;
                
                //if the user pressed like, add the active class to the like button and remove it from the dislike button
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