document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const clearBtn = document.getElementById('clear-btn');
    const resultsContainer = document.getElementById('results-container');
    const resultsCount = document.getElementById('results-count');
    const loader = document.getElementById('loader');
    const errorMessage = document.getElementById('error-message');
    const loadMoreBtn = document.getElementById('load-more-btn');

    let debounceTimeout = null;
    let currentSearchTerm = '';
    let activeAbortController = null;

    let allRecords = [];
    let renderedCount = 0;
    const batchSize = 20;

    // Lógica para detectar digitação com debounce (300ms)
    searchInput.addEventListener('input', () => {
        const query = searchInput.value.trim();
        
        // Controla exibição do botão limpar
        clearBtn.hidden = query.length === 0;

        if (query === currentSearchTerm) return;
        
        clearTimeout(debounceTimeout);

        if (query.length === 0) {
            currentSearchTerm = '';
            if (activeAbortController) {
                activeAbortController.abort();
                activeAbortController = null;
            }
            showEmptyState();
            return;
        }

        debounceTimeout = setTimeout(() => {
            performSearch(query);
        }, 300);
    });

    // Limpar busca
    clearBtn.addEventListener('click', () => {
        searchInput.value = '';
        clearBtn.hidden = true;
        currentSearchTerm = '';
        if (activeAbortController) {
            activeAbortController.abort();
            activeAbortController = null;
        }
        searchInput.focus();
        showEmptyState();
    });

    // Executar busca ao apertar Enter
    document.getElementById('search-form').addEventListener('submit', (e) => {
        e.preventDefault();
        const query = searchInput.value.trim();
        if (query.length > 0) {
            clearTimeout(debounceTimeout);
            performSearch(query);
        }
    });

    // Realiza a requisição assíncrona para a API Go
    async function performSearch(query) {
        if (activeAbortController) {
            activeAbortController.abort();
        }
        
        activeAbortController = new AbortController();
        const signal = activeAbortController.signal;

        currentSearchTerm = query;
        showLoading(true);
        hideError();
        loadMoreBtn.hidden = true;

        try {
            const sanitizedQuery = encodeURIComponent(query);
            const response = await fetch(`/api/search?q=${sanitizedQuery}`, { signal });
            
            if (!response.ok) {
                const errData = await response.json().catch(() => ({}));
                throw new Error(errData.error || `Erro no servidor: ${response.status}`);
            }

            allRecords = await response.json();
            renderedCount = 0;
            resultsContainer.replaceChildren();

            if (!allRecords || allRecords.length === 0) {
                resultsCount.textContent = '0 resultados';
                resultsCount.hidden = false;
                resultsContainer.appendChild(createEmptyState('Nenhum resultado encontrado para a busca', 'alert'));
                updateActiveTab('');
                loadMoreBtn.hidden = true;
                return;
            }

            // Destaca a aba lateral com base no primeiro resultado retornado
            if (allRecords[0] && allRecords[0].first_name) {
                updateActiveTab(allRecords[0].first_name);
            } else {
                updateActiveTab('');
            }

            resultsCount.hidden = false;
            renderNextBatch();
        } catch (error) {
            if (error.name === 'AbortError') {
                return;
            }
            console.error('Search error:', error);
            showError(error.message);
            clearResults();
            resultsCount.hidden = true;
            loadMoreBtn.hidden = true;
        } finally {
            showLoading(false);
        }
    }

    // Renderiza o próximo lote de resultados (Lazy Rendering para evitar lentidão no DOM)
    function renderNextBatch() {
        const nextBatch = allRecords.slice(renderedCount, renderedCount + batchSize);
        
        nextBatch.forEach(record => {
            const card = document.createElement('div');
            card.className = 'result-card';

            const cardHeader = document.createElement('div');
            cardHeader.className = 'card-header';

            const cardId = document.createElement('span');
            cardId.className = 'card-id';
            cardId.textContent = `#${record.combination_num}`;

            const cardCombNum = document.createElement('span');
            cardCombNum.className = 'card-comb-num';
            cardCombNum.textContent = `Comb. ID: ${record.combination_num}`;

            cardHeader.appendChild(cardId);
            cardHeader.appendChild(cardCombNum);

            const cardName = document.createElement('h3');
            cardName.className = 'card-name';
            
            let fullName = record.first_name;
            if (record.middle_name) {
                fullName += ` ${record.middle_name}`;
            }
            if (record.last_name) {
                fullName += ` ${record.last_name}`;
            }
            cardName.textContent = fullName; // Protege contra XSS

            const cardCombination = document.createElement('div');
            cardCombination.className = 'card-combination';
            cardCombination.textContent = record.combination || 'Sem combinação'; // Protege contra XSS

            card.appendChild(cardHeader);
            card.appendChild(cardName);
            card.appendChild(cardCombination);

            resultsContainer.appendChild(card);
        });

        renderedCount += nextBatch.length;
        
        resultsCount.textContent = `${renderedCount} de ${allRecords.length} ${allRecords.length === 1 ? 'resultado' : 'resultados'}`;
        
        loadMoreBtn.hidden = renderedCount >= allRecords.length;
    }

    function showEmptyState() {
        clearResults();
        resultsContainer.appendChild(createEmptyState('Digite algo acima para iniciar a pesquisa', 'search'));
        resultsCount.hidden = true;
        loadMoreBtn.hidden = true;
        hideError();
        updateActiveTab('');
    }

    function clearResults() {
        resultsContainer.replaceChildren();
        allRecords = [];
        renderedCount = 0;
    }

    function createEmptyState(message, iconType) {
        const emptyState = document.createElement('div');
        emptyState.className = 'empty-state';

        const icon = createIcon(iconType);
        const text = document.createElement('p');
        text.textContent = message;

        emptyState.appendChild(icon);
        emptyState.appendChild(text);
        return emptyState;
    }

    function createIcon(iconType) {
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('viewBox', '0 0 24 24');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('stroke-width', '1.5');
        svg.setAttribute('stroke-linecap', 'round');
        svg.setAttribute('stroke-linejoin', 'round');

        if (iconType === 'alert') {
            appendSvgElement(svg, 'circle', { cx: '12', cy: '12', r: '10' });
            appendSvgElement(svg, 'line', { x1: '12', y1: '8', x2: '12', y2: '12' });
            appendSvgElement(svg, 'line', { x1: '12', y1: '16', x2: '12.01', y2: '16' });
            return svg;
        }

        appendSvgElement(svg, 'path', { d: 'M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z' });
        return svg;
    }

    function appendSvgElement(svg, tagName, attributes) {
        const element = document.createElementNS('http://www.w3.org/2000/svg', tagName);
        Object.entries(attributes).forEach(([name, value]) => {
            element.setAttribute(name, value);
        });
        svg.appendChild(element);
    }

    function showLoading(isLoading) {
        loader.hidden = !isLoading;
    }

    function showError(msg) {
        errorMessage.textContent = msg;
        errorMessage.hidden = false;
    }

    function hideError() {
        errorMessage.hidden = true;
    }

    // Registra clique no botão "Carregar mais"
    loadMoreBtn.addEventListener('click', () => {
        renderNextBatch();
    });

    const tabs = document.querySelectorAll('.atom-tab');

    function updateActiveTab(name) {
        tabs.forEach(tab => tab.classList.remove('active-tab'));

        if (!name) {
            if (tabs.length > 0) tabs[0].classList.add('active-tab');
            return;
        }

        const firstChar = name.charAt(0).toUpperCase();
        let activeIndex = 0;

        if ('ABCD'.includes(firstChar)) {
            activeIndex = 0;
        } else if ('EFGH'.includes(firstChar)) {
            activeIndex = 1;
        } else if ('IJKL'.includes(firstChar)) {
            activeIndex = 2;
        } else if ('MNOP'.includes(firstChar)) {
            activeIndex = 3;
        } else if ('QRST'.includes(firstChar)) {
            activeIndex = 4;
        } else if ('UVWXYZ'.includes(firstChar)) {
            activeIndex = 5;
        } else {
            activeIndex = 0;
        }

        if (tabs[activeIndex]) {
            tabs[activeIndex].classList.add('active-tab');
        }
    }
});
