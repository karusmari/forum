document.addEventListener('DOMContentLoaded', function() {
    const filterForm = document.querySelector('.filter-form');
    const filterInputs = filterForm.querySelectorAll('select, input[type="checkbox"]');

    // Автоматически отправляем форму при изменении любого фильтра
    filterInputs.forEach(input => {
        input.addEventListener('change', () => {
            filterForm.submit();
        });
    });

    // Добавляем обработчик для кнопки сброса фильтров
    const resetButton = document.createElement('button');
    resetButton.type = 'button';
    resetButton.textContent = 'Reset Filters';
    resetButton.className = 'reset-filters';
    resetButton.onclick = () => {
        // Сбрасываем все фильтры
        filterForm.querySelector('select').value = '';
        filterForm.querySelectorAll('input[type="checkbox"]').forEach(checkbox => {
            checkbox.checked = false;
        });
        // Отправляем форму
        filterForm.submit();
    };

    // Добавляем кнопку сброса после кнопки применения фильтров
    filterForm.querySelector('button[type="submit"]').after(resetButton);
}); 