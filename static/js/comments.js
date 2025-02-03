function toggleEditComment(commentId) {
    const contentDiv = document.getElementById(`comment-content-${commentId}`);
    const editForm = document.getElementById(`edit-form-${commentId}`);
    
    if (editForm.style.display === 'none') {
        contentDiv.style.display = 'none';
        editForm.style.display = 'block';
    } else {
        contentDiv.style.display = 'block';
        editForm.style.display = 'none';
    }
} 