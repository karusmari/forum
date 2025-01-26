document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('.reactions button').forEach(button => {
        button.addEventListener('click', function(e) {
            e.preventDefault();
            
            const postId = this.dataset.postId;
            const type = this.dataset.type;
            
            fetch('/api/react', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    post_id: postId,
                    type: type
                })
            })
            .then(response => {
                if (response.ok) {
                    // Update UI
                    this.classList.toggle('active');
                    location.reload(); // Temporary solution, better to update only counters
                }
            })
            .catch(error => console.error('Error:', error));
        });
    });
}); 