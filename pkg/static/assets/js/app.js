(function() {
    let debounceTimeout;
    const searchInput = document.getElementById('searchInput');
    const autocompleteDiv = document.getElementById('autocomplete');

    if (searchInput && autocompleteDiv) {
        function showAutocomplete(results) {
            if (results.length === 0) {
                autocompleteDiv.classList.add('hidden');
                return;
            }

            autocompleteDiv.innerHTML = results.map(result => 
                `<div class="px-4 py-3 hover:bg-blue-50 dark:hover:bg-gray-700 cursor-pointer border-b border-gray-100 dark:border-gray-600 last:border-b-0 transition-colors duration-150" onclick="selectResult('${result.name}', '${result.path}')">
                    <div class="font-semibold text-gray-900 dark:text-white mb-1">${result.name}</div>
                    <div class="text-sm text-gray-500 dark:text-gray-400">Last updated: ${result.lastUpdatedAt}</div>
                </div>`
            ).join('');
            
            autocompleteDiv.classList.remove('hidden');
        }

        function hideAutocomplete() {
            setTimeout(() => {
                autocompleteDiv.classList.add('hidden');
            }, 150);
        }

        window.selectResult = function(name, path) {
            window.location.href = path;
        };

        searchInput.addEventListener('input', function(e) {
            const query = e.target.value.trim();
            
            clearTimeout(debounceTimeout);
            
            if (query.length < 2) {
                autocompleteDiv.classList.add('hidden');
                return;
            }

            debounceTimeout = setTimeout(() => {
                fetch(`/api/search?q=${encodeURIComponent(query)}`)
                    .then(response => response.json())
                    .then(data => {
                        showAutocomplete(data.results.slice(0, 5)); // Show max 5 suggestions
                    })
                    .catch(error => {
                        console.error('Search error:', error);
                        autocompleteDiv.classList.add('hidden');
                    });
            }, 300);
        });

        searchInput.addEventListener('blur', hideAutocomplete);
        searchInput.addEventListener('focus', function (e) {
            const query = e.target.value.trim();
            if (query.length >= 2) {
                // Trigger search again on focus if there's already a query
                searchInput.dispatchEvent(new Event('input'));
            }
        });

        // Handle keyboard navigation
        let currentSelection = -1;
        searchInput.addEventListener('keydown', function(e) {
            const items = autocompleteDiv.querySelectorAll('div[onclick]');
            
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                currentSelection = Math.min(currentSelection + 1, items.length - 1);
                updateSelection(items);
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                currentSelection = Math.max(currentSelection - 1, -1);
                updateSelection(items);
            } else if (e.key === 'Enter' && currentSelection >= 0) {
                e.preventDefault();
                items[currentSelection].click();
            } else if (e.key === 'Escape') {
                autocompleteDiv.classList.add('hidden');
                currentSelection = -1;
            }
        });

        function updateSelection(items) {
            items.forEach((item, index) => {
                if (index === currentSelection) {
                    item.classList.add('bg-blue-100', 'dark:bg-gray-600');
                    item.classList.remove('hover:bg-blue-50', 'dark:hover:bg-gray-700');
                } else {
                    item.classList.remove('bg-blue-100', 'dark:bg-gray-600');
                    item.classList.add('hover:bg-blue-50', 'dark:hover:bg-gray-700');
                }
            });
        }
    }

    // Repository scan functionality
    window.fetchScan = function(url, image) {
        fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: image,
            redirect: 'follow'
        })
            .then(response => {
                if (response.redirected) {
                    window.open(response.url, '_blank', 'noopener,noreferrer');
                } else {
                    return response.json();
                }
            })
            .catch(error => {
                console.error('Error:', error);
                alert("Error: " + error.message);
            });
    }

    // Dark mode toggle functionality
    function initDarkModeToggle() {
        const darkModeToggle = document.getElementById('darkModeToggle');
        if (!darkModeToggle) return;

        darkModeToggle.addEventListener('click', () => {
            const isDark = document.documentElement.classList.toggle('dark');
            localStorage.setItem('theme', isDark ? 'dark' : 'light');
        });
    }

    // Initialize dark mode toggle when DOM is loaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initDarkModeToggle);
        return;
    }
    initDarkModeToggle();
})();