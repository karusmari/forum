function handleBack() {
    const urlParams = new URLSearchParams(window.location.search);
    const categoryId = urlParams.get('cat');
    
    if (categoryId) {
        window.location.href = '/category/' + categoryId;
        return;
    }
    
    if (window.history.length > 1) {
        window.history.back();
    } else {
        window.location.href = '/';
    }
} 