function validateComment(form) {
    const content = form.querySelector('textarea[name="content"]').value.trim();
    if (content === '') {
        alert('Comment cannot be empty');
        return false;
    }
    return true;
}
