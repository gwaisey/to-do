// public/app.js

const API_BASE = ""; // Local relative URL works since we are serving this from the same host
let currentToken = localStorage.getItem("todo_api_token") || "";
let currentUser = null;
try {
    currentUser = JSON.parse(localStorage.getItem("todo_api_user")) || null;
} catch (e) {
    currentUser = null;
}

// Active filters state
let activeStatus = "";
let activePriority = "";
let activeTag = "";
let activeSortBy = "created_at";
let activeSortOrder = "desc";
let searchTimeout = null;
let currentTodos = [];
let selectedTodoId = null;

// Selection Mode State
let isSelectionMode = false;
let selectedTodos = new Set();

// Pagination State
let currentPage = 1;
const limitPerPage = 10;
let hasMoreTodos = false;
let isLoadingMore = false;
let hasAnyTodos = false;

// DOM Elements
const authScreen = document.getElementById("auth-screen");
const dashboardScreen = document.getElementById("dashboard-screen");
const dashboardContent = document.getElementById("dashboard-content");
const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");
const toRegisterLink = document.getElementById("to-register");
const toLoginLink = document.getElementById("to-login");
const authSubtitle = document.getElementById("auth-subtitle");
const displayUsername = document.getElementById("display-username");
const logoutBtn = document.getElementById("logout-btn");
const searchInput = document.getElementById("search-input");
const priorityFilter = document.getElementById("priority-filter");
const tagFilter = document.getElementById("tag-filter");
const sortFilter = document.getElementById("sort-filter");
const todoList = document.getElementById("todo-list");
const selectModeBtn = document.getElementById("select-mode-btn");
const bulkActionBar = document.getElementById("bulk-action-bar");
const bulkSelectCount = document.getElementById("bulk-select-count");
const bulkCompleteBtn = document.getElementById("bulk-complete-btn");
const bulkDeleteBtn = document.getElementById("bulk-delete-btn");

// Header and navigation
const themeToggleBtn = document.getElementById("theme-toggle-btn");
const themeActionLabel = document.getElementById("theme-action-label");
const profileMenuTrigger = document.getElementById("profile-menu-trigger");
const profileMenu = document.getElementById("profile-menu");
const sidebarToggleBtn = document.getElementById("sidebar-toggle-btn");
const sidebarCloseBtn = document.getElementById("sidebar-close-btn");
const sidebarBackdrop = document.getElementById("sidebar-backdrop");
const detailBackBtn = document.getElementById("detail-back-btn");
const detailPanel = document.getElementById("todo-detail-panel");
const detailEmptyHint = document.getElementById("detail-empty-hint");

// Infinite Scroll elements
const scrollSentinel = document.getElementById("scroll-sentinel");
const loadingSpinner = document.getElementById("loading-spinner");

// Stats DOM
const statTotal = document.getElementById("stat-total");
const statPending = document.getElementById("stat-pending");
const statInProgress = document.getElementById("stat-in-progress");
const statCompleted = document.getElementById("stat-completed");
const statusFilters = document.querySelectorAll(".status-filter");

// Modal DOM
const todoModal = document.getElementById("todo-modal");
const addTodoFab = document.getElementById("add-todo-fab");
const cancelTodoBtn = document.getElementById("cancel-todo-btn");
const saveTodoBtn = document.getElementById("save-todo-btn");
const todoForm = document.getElementById("todo-form");
const modalTitle = document.getElementById("modal-title");
const editTodoId = document.getElementById("edit-todo-id");
const todoTitleInput = document.getElementById("todo-title");
const todoDescInput = document.getElementById("todo-desc");
const todoDueDateInput = document.getElementById("todo-duedate");
const todoTagsInput = document.getElementById("todo-tags");
const modalSubtasksContainer = document.getElementById("modal-subtasks-container");
const addSubtaskBtn = document.getElementById("add-subtask-btn");
const detailSubtasksCard = document.getElementById("detail-subtasks-card");
const detailSubtasksContainer = document.getElementById("detail-subtasks-container");
let modalSubtasks = [];
let modalReturnFocus = null;

// ── UTILITIES ──

// Show custom toast message
function showToast(message, isSuccess = true) {
    const toast = document.getElementById("toast");
    toast.innerText = message;
    toast.style.borderLeft = `4px solid ${isSuccess ? "var(--success)" : "var(--danger)"}`;
    toast.classList.add("show");
    setTimeout(() => {
        toast.classList.remove("show");
    }, 3000);
}

// Format remaining time or overdue duration in Indonesian
function formatRemainingTime(dueDateStr, status) {
    if (!dueDateStr) return "";
    
    const dueDate = new Date(dueDateStr);
    const now = new Date();
    
    const isDone = status === "done";
    const diffMs = dueDate - now;
    const isOverdue = diffMs < 0;
    
    // If completed, we don't display remaining or overdue info
    if (isDone) return "";
    
    const absDiffMs = Math.abs(diffMs);
    const diffMins = Math.floor(absDiffMs / (1000 * 60));
    const diffHours = Math.floor(absDiffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(absDiffMs / (1000 * 60 * 60 * 24));
    
    let timeText = "";
    if (diffDays > 0) {
        timeText = `${diffDays} hari`;
    } else if (diffHours > 0) {
        const minsRemaining = diffMins % 60;
        timeText = `${diffHours} jam${minsRemaining > 0 ? ` ${minsRemaining} menit` : ""}`;
    } else {
        timeText = `${diffMins} menit`;
    }
    
    return isOverdue ? `Terlambat ${timeText}` : `Sisa ${timeText}`;
}

// Fetch helper that automatically adds JWT header
async function request(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    
    // Add auth header if we have a token
    const headers = {
        "Content-Type": "application/json",
        ...(options.headers || {})
    };
    
    if (currentToken) {
        headers["Authorization"] = `Bearer ${currentToken}`;
    }
    
    const config = {
        ...options,
        headers
    };
    
    try {
        const response = await fetch(url, config);
        
        // Handle unauthorized token (expired/invalid)
        if (response.status === 401) {
            handleLogout();
            showToast("Sesi habis, silakan login kembali.", false);
            return null;
        }
        
        const data = await response.json();
        if (!response.ok) {
            throw new Error(data.message || data.error || "Terjadi kesalahan server");
        }
        
        return data;
    } catch (err) {
        showToast(err.message, false);
        console.error("API Request Error:", err);
        throw err;
    }
}

// ── AUTHENTICATION ──

function checkAuth() {
    if (currentToken && currentUser) {
        authScreen.classList.remove("active");
        dashboardScreen.classList.add("active");
        displayUsername.innerText = currentUser.username;
        document.getElementById("user-avatar").innerText = currentUser.username.charAt(0).toUpperCase();
        loadTodos();
    } else {
        dashboardScreen.classList.remove("active");
        authScreen.classList.add("active");
    }
}

// Switch between Register and Login forms
toRegisterLink.addEventListener("click", (e) => {
    e.preventDefault();
    loginForm.classList.remove("active");
    registerForm.classList.add("active");
    authSubtitle.innerText = "Daftar untuk membuat akun baru";
});

toLoginLink.addEventListener("click", (e) => {
    e.preventDefault();
    registerForm.classList.remove("active");
    loginForm.classList.add("active");
    authSubtitle.innerText = "Masuk untuk melihat daftar tugas Anda";
});

// Handle Login Form Submit
loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const email = document.getElementById("login-email").value.trim();
    const password = document.getElementById("login-password").value;
    
    try {
        const res = await request("/login", {
            method: "POST",
            body: JSON.stringify({ email, password })
        });
        
        if (res && res.success) {
            currentToken = res.data.token;
            currentUser = res.data.user;
            localStorage.setItem("todo_api_token", currentToken);
            localStorage.setItem("todo_api_user", JSON.stringify(currentUser));
            showToast("Login Berhasil! Selamat datang.");
            checkAuth();
            loginForm.reset();
        }
    } catch (err) {
        // Error already handled by toast in request helper
    }
});

// Handle Register Form Submit
registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const username = document.getElementById("reg-username").value.trim();
    const email = document.getElementById("reg-email").value.trim();
    const password = document.getElementById("reg-password").value;
    
    try {
        const res = await request("/register", {
            method: "POST",
            body: JSON.stringify({ username, email, password })
        });
        
        if (res && res.success) {
            showToast("Pendaftaran Berhasil! Silakan masuk.");
            // Switch to login
            registerForm.reset();
            toLoginLink.click();
        }
    } catch (err) {
        // Error handled
    }
});

// Handle Logout
function handleLogout() {
    currentToken = "";
    currentUser = null;
    localStorage.removeItem("todo_api_token");
    localStorage.removeItem("todo_api_user");
    checkAuth();
}

logoutBtn.addEventListener("click", handleLogout);

// ── TODO ACTIONS & RENDERING ──

// Load Todos from API
// Load Todos from API (Halaman 1)
async function loadTodos() {
    currentPage = 1;
    isLoadingMore = false;
    
    let queryParams = [];
    if (activeStatus) queryParams.push(`status=${activeStatus}`);
    if (activePriority) queryParams.push(`priority=${activePriority}`);
    if (activeTag) queryParams.push(`tag=${encodeURIComponent(activeTag)}`);
    if (activeSortBy) queryParams.push(`sort_by=${activeSortBy}`);
    if (activeSortOrder) queryParams.push(`order=${activeSortOrder}`);
    
    const searchVal = searchInput.value.trim();
    if (searchVal) {
        queryParams.push(`search=${encodeURIComponent(searchVal)}`);
    }
    
    // Kirim parameter paginasi
    queryParams.push(`page=${currentPage}`);
    queryParams.push(`limit=${limitPerPage}`);
    
    const queryString = queryParams.length ? `?${queryParams.join("&")}` : "";
    
    try {
        todoList.setAttribute("aria-busy", "true");
        const res = await request(`/todos${queryString}`);
        if (res && res.success) {
            currentTodos = res.data.todos || [];
            
            // Render ulang dari awal
            updateStats(res.data.stats);
            renderTodoList(currentTodos);
            
            // Sinkronisasi status has_more
            const pag = res.data.pagination;
            hasMoreTodos = pag ? pag.has_more : false;
            
            if (scrollSentinel) {
                scrollSentinel.style.display = hasMoreTodos ? "flex" : "none";
            }
            if (loadingSpinner) {
                loadingSpinner.hidden = true;
            }
            
            // Sync active selection
            if (selectedTodoId) {
                const stillExists = currentTodos.some(t => t.id === selectedTodoId);
                if (stillExists) {
                    selectTodo(selectedTodoId);
                } else {
                    clearSelection();
                }
            } else {
                clearSelection();
            }
        }
    } catch (err) {
        todoList.innerHTML = `<div class="empty-state"><h2>Daftar tugas belum dapat dimuat</h2><p>Periksa koneksi lalu coba lagi.</p></div>`;
    } finally {
        todoList.setAttribute("aria-busy", "false");
    }
}

// Load More Todos dari API (Halaman > 1)
async function loadMoreTodos() {
    if (isLoadingMore || !hasMoreTodos) return;
    
    isLoadingMore = true;
    if (loadingSpinner) {
        loadingSpinner.hidden = false;
    }
    
    currentPage++;
    
    let queryParams = [];
    if (activeStatus) queryParams.push(`status=${activeStatus}`);
    if (activePriority) queryParams.push(`priority=${activePriority}`);
    if (activeTag) queryParams.push(`tag=${encodeURIComponent(activeTag)}`);
    if (activeSortBy) queryParams.push(`sort_by=${activeSortBy}`);
    if (activeSortOrder) queryParams.push(`order=${activeSortOrder}`);
    
    const searchVal = searchInput.value.trim();
    if (searchVal) {
        queryParams.push(`search=${encodeURIComponent(searchVal)}`);
    }
    
    queryParams.push(`page=${currentPage}`);
    queryParams.push(`limit=${limitPerPage}`);
    
    const queryString = queryParams.length ? `?${queryParams.join("&")}` : "";
    
    try {
        const res = await request(`/todos${queryString}`);
        if (res && res.success) {
            const newTodos = res.data.todos || [];
            currentTodos = currentTodos.concat(newTodos);
            
            // Tambahkan item baru ke daftar yang sudah ada
            appendTodoList(newTodos);
            updateStats(res.data.stats);
            
            const pag = res.data.pagination;
            hasMoreTodos = pag ? pag.has_more : false;
            
            if (scrollSentinel) {
                scrollSentinel.style.display = hasMoreTodos ? "flex" : "none";
            }
        } else {
            currentPage--;
        }
    } catch (err) {
        currentPage--;
    } finally {
        isLoadingMore = false;
        if (loadingSpinner) {
            loadingSpinner.hidden = true;
        }
    }
}

function svgIcon(name) {
    const paths = {
        calendar: '<rect x="4" y="5" width="16" height="15" rx="2"></rect><path d="M8 3v4m8-4v4M4 9h16"></path>',
        check: '<path d="m5 12 4 4L19 6"></path>',
        edit: '<path d="M12 20h9"></path><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"></path>',
        play: '<path d="M8 5v14l11-7z"></path>',
        pause: '<rect x="6" y="4" width="4" height="16" rx="1"></rect><rect x="14" y="4" width="4" height="16" rx="1"></rect>',
        trash: '<path d="M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m4 4v6m6-6v6"></path>'
    };
    return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || ""}</svg>`;
}

function createTodoElement(todo, searchQuery) {
    const isSelected = selectedTodos.has(todo.id);
    const isDone = todo.status === "done";
    const todoEl = document.createElement("article");
    todoEl.className = `todo-item ${isDone ? "done" : ""} ${todo.status === "in_progress" ? "in-progress" : ""} ${isSelected ? "selected-item" : ""}`;
    todoEl.dataset.id = todo.id;
    todoEl.tabIndex = 0;
    todoEl.setAttribute("aria-label", `Tugas: ${todo.title}`);

    let dueDateHTML = "";
    if (todo.due_date) {
        const dateObj = new Date(todo.due_date);
        const isOverdue = dateObj < new Date() && !isDone;
        const formattedDate = dateObj.toLocaleDateString("id-ID", {
            day: "numeric",
            month: "short",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit"
        });
        const remainingText = formatRemainingTime(todo.due_date, todo.status);
        dueDateHTML = `<span class="due-date ${isOverdue ? "overdue" : ""}">${svgIcon("calendar")}<span>${escapeHTML(formattedDate)}${remainingText ? ` (${escapeHTML(remainingText)})` : ""}</span></span>`;
    }

    const tagsHTML = (todo.tags || [])
        .map(tag => `<span class="tag-badge">${escapeHTML(String(tag))}</span>`)
        .join("");
    const priorityLabels = { low: "Rendah", medium: "Sedang", high: "Tinggi" };
    const priorityClass = Object.prototype.hasOwnProperty.call(priorityLabels, todo.priority) ? todo.priority : "medium";
    const priorityHTML = `<span class="priority-pill ${priorityClass}">${priorityLabels[priorityClass]}</span>`;

    let subTaskProgressHTML = "";
    let subTaskProgressBarHTML = "";
    if (todo.sub_tasks && todo.sub_tasks.length > 0) {
        const totalSub = todo.sub_tasks.length;
        const completedSub = todo.sub_tasks.filter(st => st.status === "done").length;
        const pct = Math.round((completedSub / totalSub) * 100);
        subTaskProgressHTML = `<span class="subtask-progress-badge">${svgIcon("check")} ${completedSub}/${totalSub}</span>`;
        subTaskProgressBarHTML = `
            <div class="subtask-progress-container" aria-label="Progres sub-tugas ${pct}%">
                <div class="subtask-progress-track" aria-hidden="true">
                    <div class="subtask-progress-fill ${pct === 100 ? "complete" : ""}" style="width: ${pct}%"></div>
                </div>
                <span class="subtask-progress-label">${pct}%</span>
            </div>
        `;
    }

    const checkLabel = isSelectionMode
        ? `Pilih ${todo.title}`
        : `${isDone ? "Tandai belum selesai" : "Tandai selesai"}: ${todo.title}`;
    const progressButtonHTML = todo.status === "pending"
        ? `<button class="icon-button play" type="button" title="Mulai proses" aria-label="Mulai proses: ${escapeHTML(todo.title)}">${svgIcon("play")}</button>`
        : todo.status === "in_progress"
            ? `<button class="icon-button pause" type="button" title="Tunda tugas" aria-label="Tunda tugas: ${escapeHTML(todo.title)}">${svgIcon("pause")}</button>`
            : "";

    todoEl.innerHTML = `
        <div class="todo-check">
            <button class="checkbox-circle ${isDone ? "checked" : ""} ${isSelected ? "selected" : ""}" type="button" aria-label="${escapeHTML(checkLabel)}" aria-pressed="${isSelectionMode ? isSelected : isDone}" title="${escapeHTML(checkLabel)}"></button>
        </div>
        <div class="todo-body">
            <div class="todo-title-row">
                <h3 class="todo-item-title">${highlightText(todo.title, searchQuery)}</h3>
                <div class="todo-actions">
                    ${progressButtonHTML}
                    <button class="icon-button edit" type="button" title="Edit tugas" aria-label="Edit tugas: ${escapeHTML(todo.title)}">${svgIcon("edit")}</button>
                    <button class="icon-button danger delete" type="button" title="Hapus tugas" aria-label="Hapus tugas: ${escapeHTML(todo.title)}">${svgIcon("trash")}</button>
                </div>
            </div>
            ${todo.description ? `<p class="todo-item-desc">${highlightText(todo.description, searchQuery)}</p>` : ""}
            ${subTaskProgressBarHTML}
            <div class="todo-meta">
                ${priorityHTML}
                ${subTaskProgressHTML}
                ${dueDateHTML}
                ${tagsHTML}
            </div>
        </div>
    `;

    const checkbox = todoEl.querySelector(".checkbox-circle");
    checkbox.addEventListener("click", event => handleCheckboxClick(event, todo.id, todo.status));

    const progressButton = todoEl.querySelector(".todo-actions .play, .todo-actions .pause");
    if (progressButton) {
        progressButton.addEventListener("click", () => toggleProgress(todo.id, todo.status));
    }
    todoEl.querySelector(".todo-actions .edit").addEventListener("click", () => openEditModal(todo.id));
    todoEl.querySelector(".todo-actions .delete").addEventListener("click", () => deleteTodo(todo.id));

    todoEl.addEventListener("click", event => {
        if (event.target.closest("button")) return;
        if (isSelectionMode) {
            toggleTodoSelection(todo.id);
        } else {
            selectTodo(todo.id);
        }
    });
    todoEl.addEventListener("keydown", event => {
        if (event.target !== todoEl || (event.key !== "Enter" && event.key !== " ")) return;
        event.preventDefault();
        if (isSelectionMode) {
            toggleTodoSelection(todo.id);
        } else {
            selectTodo(todo.id);
        }
    });

    return todoEl;
}

function renderTodoList(todos) {
    if (!todos || todos.length === 0) {
        const title = hasAnyTodos ? "Tidak ada tugas yang cocok" : "Belum ada tugas";
        const description = hasAnyTodos
            ? "Coba ubah kata pencarian atau filter yang dipilih."
            : "Tambahkan tugas pertama agar semua pengingatmu tersusun rapi.";
        todoList.innerHTML = `
            <div class="empty-state">
                <span class="empty-state-icon" aria-hidden="true">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="17" rx="2"></rect><path d="M8 9h8M8 13h5"></path><path d="M9 2v4m6-4v4"></path></svg>
                </span>
                <h2>${title}</h2>
                <p>${description}</p>
                ${hasAnyTodos ? "" : `<button id="empty-create-btn" class="btn btn-primary" type="button"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14"></path></svg>Buat tugas pertama</button>`}
            </div>
        `;
        return;
    }

    todoList.innerHTML = "";
    appendTodoList(todos);
}

function appendTodoList(todos) {
    if (!todos) return;
    const searchQuery = searchInput ? searchInput.value.trim() : "";
    todos.forEach(todo => todoList.appendChild(createTodoElement(todo, searchQuery)));
}

function updateStats(stats) {
    if (!stats) return;

    hasAnyTodos = Number(stats.total) > 0;
    statTotal.textContent = stats.total || 0;
    statPending.textContent = stats.pending || 0;
    statInProgress.textContent = stats.in_progress || 0;
    statCompleted.textContent = stats.completed || 0;

    statusFilters.forEach(button => {
        const isActive = button.dataset.status === activeStatus;
        button.classList.toggle("active", isActive);
        button.setAttribute("aria-pressed", String(isActive));
    });

    if (stats.tags) {
        const previousSelection = activeTag;
        tagFilter.innerHTML = '<option value="">Semua Tag</option>';
        stats.tags.forEach(tag => {
            const option = document.createElement("option");
            option.value = tag;
            option.textContent = tag;
            option.selected = tag === previousSelection;
            tagFilter.appendChild(option);
        });
    }
}

// Toggle Todo Completed via API
async function toggleComplete(id, currentStatus) {
    const isDone = currentStatus === "done";
    
    try {
        if (!isDone) {
            // Use PATCH endpoint
            const res = await request(`/todos/${id}/complete`, { method: "PATCH" });
            if (res && res.success) {
                showToast("Tugas ditandai selesai!");
                loadTodos();
            }
        } else {
            // Revert back to pending using PUT update endpoint
            const res = await request(`/todos/${id}`, {
                method: "PUT",
                body: JSON.stringify({ status: "pending" })
            });
            if (res && res.success) {
                showToast("Tugas dikembalikan ke proses.");
                loadTodos();
            }
        }
    } catch (e) {
        // handled
    }
}

// Toggle Todo Progress (pending <=> in_progress) via API
async function toggleProgress(id, currentStatus) {
    const newStatus = currentStatus === "pending" ? "in_progress" : "pending";
    try {
        const res = await request(`/todos/${id}`, {
            method: "PUT",
            body: JSON.stringify({ status: newStatus })
        });
        if (res && res.success) {
            showToast(newStatus === "in_progress" ? "Tugas mulai diproses!" : "Tugas ditunda.");
            loadTodos();
        }
    } catch (e) {
        // handled
    }
}

// Delete Todo via API
async function deleteTodo(id) {
    if (!confirm("Apakah Anda yakin ingin menghapus tugas ini?")) return;
    
    try {
        const res = await request(`/todos/${id}`, { method: "DELETE" });
        if (res && res.success) {
            showToast("Tugas berhasil dihapus.");
            loadTodos();
        }
    } catch (e) {
        // handled
    }
}

// Toggle Sub-task Completion status in Detail Panel with Auto-save to server
async function toggleSubTaskStatus(todoId, subTaskId) {
    const todo = currentTodos.find(t => t.id === todoId);
    if (!todo) return;
    
    const subTasks = todo.sub_tasks.map(st => {
        if (st.id === subTaskId) {
            return {
                ...st,
                status: st.status === 'done' ? 'pending' : 'done'
            };
        }
        return st;
    });
    
    try {
        const res = await request(`/todos/${todoId}`, {
            method: "PUT",
            body: JSON.stringify({ sub_tasks: subTasks })
        });
        if (res && res.success) {
            // Update the local todo object
            const index = currentTodos.findIndex(t => t.id === todoId);
            if (index !== -1) {
                currentTodos[index] = res.data;
            }
            renderTodoList(currentTodos);
            selectTodo(todoId);
        }
    } catch (e) {
        showToast("Gagal memperbarui status sub-tugas.", false);
    }
}

// Render sub-tasks inside Edit/Create Modal
function renderModalSubtasks() {
    if (!modalSubtasksContainer) return;
    modalSubtasksContainer.innerHTML = "";
    
    modalSubtasks.forEach((st, index) => {
        const itemEl = document.createElement("div");
        itemEl.className = "modal-subtask-item";
        itemEl.innerHTML = `
            <label class="sr-only" for="modal-subtask-${index}">Nama sub-tugas</label>
            <input type="text" id="modal-subtask-${index}" placeholder="Nama sub-tugas..." value="${escapeHTML(st.title)}">
            <button type="button" class="btn-remove-subtask" title="Hapus sub-tugas" aria-label="Hapus sub-tugas ${index + 1}">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"></path></svg>
            </button>
        `;
        
        // Update model title on input
        const input = itemEl.querySelector("input");
        input.addEventListener("input", (e) => {
            modalSubtasks[index].title = e.target.value;
        });
        
        // Remove button handler
        const removeBtn = itemEl.querySelector(".btn-remove-subtask");
        removeBtn.addEventListener("click", () => {
            // Read current titles first to preserve any typing
            const inputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
            inputs.forEach((inp, idx) => {
                if (modalSubtasks[idx]) {
                    modalSubtasks[idx].title = inp.value;
                }
            });
            modalSubtasks.splice(index, 1);
            renderModalSubtasks();
        });
        
        modalSubtasksContainer.appendChild(itemEl);
    });
}

// ── MODAL SHEET & FORMS ──

function openCreateModal() {
    modalReturnFocus = document.activeElement;
    modalTitle.innerText = "Tugas Baru";
    editTodoId.value = "";
    todoForm.reset();
    
    // Pre-populate with current local date/time
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(9, 0, 0, 0);
    // Format to yyyy-MM-ddThh:mm
    const year = tomorrow.getFullYear();
    const month = String(tomorrow.getMonth() + 1).padStart(2, '0');
    const day = String(tomorrow.getDate()).padStart(2, '0');
    const hours = String(tomorrow.getHours()).padStart(2, '0');
    const minutes = String(tomorrow.getMinutes()).padStart(2, '0');
    todoDueDateInput.value = `${year}-${month}-${day}T${hours}:${minutes}`;

    modalSubtasks = [];
    renderModalSubtasks();

    todoModal.classList.add("active");
    todoModal.setAttribute("aria-hidden", "false");
    todoTitleInput.focus();
}

async function openEditModal(id) {
    try {
        const res = await request(`/todos/${id}`);
        if (res && res.success) {
            const todo = res.data;
            modalReturnFocus = document.activeElement;
            modalTitle.innerText = "Edit Tugas";
            editTodoId.value = todo.id;
            todoTitleInput.value = todo.title;
            todoDescInput.value = todo.description || "";
            
            // Set priority radio
            const priorityRadio = document.querySelector(`input[name="todo-priority"][value="${todo.priority}"]`);
            if (priorityRadio) priorityRadio.checked = true;
            
            // Set due date format (yyyy-MM-ddThh:mm)
            if (todo.due_date) {
                const d = new Date(todo.due_date);
                // Adjust for local timezone input
                const offset = d.getTimezoneOffset() * 60000;
                const localISOTime = (new Date(d.getTime() - offset)).toISOString().slice(0, 16);
                todoDueDateInput.value = localISOTime;
            } else {
                todoDueDateInput.value = "";
            }
            
            // Set tags
            todoTagsInput.value = todo.tags ? todo.tags.join(", ") : "";
            
            modalSubtasks = todo.sub_tasks ? JSON.parse(JSON.stringify(todo.sub_tasks)) : [];
            renderModalSubtasks();
            
            todoModal.classList.add("active");
            todoModal.setAttribute("aria-hidden", "false");
            todoTitleInput.focus();
        }
    } catch (e) {
        // handled
    }
}

function closeModal() {
    todoModal.classList.remove("active");
    todoModal.setAttribute("aria-hidden", "true");
    const focusTarget = modalReturnFocus instanceof HTMLElement && modalReturnFocus.isConnected
        ? modalReturnFocus
        : addTodoFab;
    focusTarget.focus();
}

// Save Todo (Create or Update)
todoForm.addEventListener("submit", async (e) => {
    e.preventDefault();
});

saveTodoBtn.addEventListener("click", async () => {
    const title = todoTitleInput.value.trim();
    if (!title) {
        showToast("Judul tidak boleh kosong.", false);
        return;
    }
    
    const description = todoDescInput.value.trim();
    const priority = document.querySelector('input[name="todo-priority"]:checked').value;
    
    // Parse tags: split by comma, trim spaces, remove empty tags
    const tags = todoTagsInput.value
        .split(",")
        .map(t => t.trim())
        .filter(t => t.length > 0);
        
    // Format due date to RFC3339
    let due_date = "";
    if (todoDueDateInput.value) {
        const localDate = new Date(todoDueDateInput.value);
        due_date = localDate.toISOString(); // converts to UTC ISO-8601 which net/http handles
    }
    
    // Update subtasks title from inputs before saving
    if (modalSubtasksContainer) {
        const subTaskInputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        subTaskInputs.forEach((input, index) => {
            if (modalSubtasks[index]) {
                modalSubtasks[index].title = input.value.trim();
            }
        });
    }
    
    // Filter out sub-tasks with empty titles
    const subtasksToSend = modalSubtasks.filter(st => st.title.trim().length > 0);
    
    const id = editTodoId.value;
    const isEdit = !!id;
    
    const url = isEdit ? `/todos/${id}` : "/todos";
    const method = isEdit ? "PUT" : "POST";
    
    const payload = {
        title,
        description,
        priority,
        due_date,
        tags,
        sub_tasks: subtasksToSend
    };
    

    // Actually in JS we just send normal properties, Go's json.Unmarshal maps to pointers.
    // If due_date is empty, send nil/null so it is ignored or cleared.
    const finalPayload = { ...payload };
    if (!due_date) {
        if (isEdit) {
            finalPayload.due_date = null;
        } else {
            delete finalPayload.due_date;
        }
    }
    
    try {
        const res = await request(url, {
            method,
            body: JSON.stringify(finalPayload)
        });
        
        if (res && res.success) {
            showToast(isEdit ? "Tugas berhasil diperbarui!" : "Tugas berhasil ditambahkan!");
            closeModal();
            loadTodos();
        }
    } catch (e) {
        // handled
    }
});

// ── EVENTS ──

addTodoFab.addEventListener("click", openCreateModal);
cancelTodoBtn.addEventListener("click", closeModal);
todoList.addEventListener("click", event => {
    if (event.target.closest("#empty-create-btn")) openCreateModal();
});
todoModal.addEventListener("click", event => {
    if (event.target === todoModal) closeModal();
});
detailBackBtn.addEventListener("click", clearSelection);
sidebarToggleBtn.addEventListener("click", () => setSidebarOpen(true));
sidebarCloseBtn.addEventListener("click", () => setSidebarOpen(false));
sidebarBackdrop.addEventListener("click", () => setSidebarOpen(false));
profileMenuTrigger.addEventListener("click", () => setProfileMenuOpen(profileMenu.hidden));
document.addEventListener("click", event => {
    if (!event.target.closest(".user-menu")) setProfileMenuOpen(false);
});

if (addSubtaskBtn) {
    addSubtaskBtn.addEventListener("click", () => {
        // First, read current input values from DOM to preserve user typing
        const inputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        inputs.forEach((inp, idx) => {
            if (modalSubtasks[idx]) {
                modalSubtasks[idx].title = inp.value;
            }
        });
        
        // Add a new empty subtask
        modalSubtasks.push({
            id: 'sub_' + Math.random().toString(36).substr(2, 9),
            title: "",
            status: "pending"
        });
        
        renderModalSubtasks();
        
        // Focus on the newly added input
        const newInputs = modalSubtasksContainer.querySelectorAll(".modal-subtask-item input");
        if (newInputs.length > 0) {
            newInputs[newInputs.length - 1].focus();
        }
    });
}

statusFilters.forEach(button => {
    button.addEventListener("click", () => {
        activeStatus = button.dataset.status;
        statusFilters.forEach(filter => {
            const isActive = filter === button;
            filter.classList.toggle("active", isActive);
            filter.setAttribute("aria-pressed", String(isActive));
        });
        setSidebarOpen(false);
        loadTodos();
    });
});

// Priority filter select handler
priorityFilter.addEventListener("change", (e) => {
    activePriority = e.target.value;
    loadTodos();
});

// Tag filter select handler
tagFilter.addEventListener("change", (e) => {
    activeTag = e.target.value;
    loadTodos();
});

// Sort filter select handler
sortFilter.addEventListener("change", (e) => {
    const parts = e.target.value.split("_");
    if (parts.length >= 2) {
        activeSortOrder = parts.pop();
        activeSortBy = parts.join("_");
    }
    loadTodos();
});

// Search input handler with debounce
searchInput.addEventListener("input", () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        loadTodos();
    }, 400); // 400ms debounce
});

// HTML escaping helper
function escapeHTML(str) {
    if (str === null || str === undefined) return "";
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Search text highlighting helper with XSS protection and case sensitivity preservation
function highlightText(text, query) {
    if (!text) return "";
    if (!query) return escapeHTML(text);
    
    // Escape regex metacharacters in query safely
    const escapedQuery = query.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
    const regex = new RegExp(`(${escapedQuery})`, "gi");
    
    const parts = text.split(regex);
    
    return parts.map(part => {
        if (part.toLowerCase() === query.toLowerCase()) {
            return `<mark class="search-highlight">${escapeHTML(part)}</mark>`;
        }
        return escapeHTML(part);
    }).join("");
}

// Master-Detail Panel Functions
function selectTodo(id) {
    const todo = currentTodos.find(t => t.id === id);
    if (!todo) return;

    selectedTodoId = id;
    dashboardContent.classList.add("has-selection");
    detailPanel.classList.toggle("mobile-open", window.innerWidth < 768);
    detailPanel.setAttribute("aria-hidden", "false");
    detailEmptyHint.hidden = true;
    setSidebarOpen(false);

    document.querySelectorAll(".todo-item").forEach(item => {
        item.classList.toggle("active-todo", item.dataset.id === id);
    });

    const emptyState = detailPanel.querySelector(".detail-empty-state");
    const detailContent = detailPanel.querySelector(".detail-content");
    emptyState.hidden = true;
    detailContent.hidden = false;
    document.getElementById("detail-title").textContent = todo.title;
    
    const statusMap = { pending: "Tunda", in_progress: "Proses", done: "Selesai" };
    const statusBadge = document.getElementById("detail-status");
    if (statusBadge) {
        statusBadge.textContent = statusMap[todo.status] || todo.status;
        statusBadge.className = `tag-badge ${todo.status === "done" ? "checked" : ""}`;
    }
    
    const priorityLabel = document.getElementById("detail-priority");
    if (priorityLabel) {
        const priorityLabels = { low: "Rendah", medium: "Sedang", high: "Tinggi" };
        const priority = Object.prototype.hasOwnProperty.call(priorityLabels, todo.priority) ? todo.priority : "medium";
        priorityLabel.textContent = priorityLabels[priority];
        priorityLabel.className = `priority-pill ${priority}`;
    }
    
    const dueDateSpan = document.getElementById("detail-duedate");
    if (dueDateSpan) {
        if (todo.due_date) {
            const dateObj = new Date(todo.due_date);
            const remainingText = formatRemainingTime(todo.due_date, todo.status);
            const displayRemaining = remainingText ? ` (${remainingText})` : "";
            dueDateSpan.textContent = dateObj.toLocaleDateString("id-ID", {
                day: "numeric",
                month: "long",
                year: "numeric",
                hour: "2-digit",
                minute: "2-digit"
            }) + displayRemaining;
        } else {
            dueDateSpan.textContent = "Tidak ada";
        }
    }
    
    // Description Card
    const descCard = document.getElementById("detail-desc-card");
    const descText = document.getElementById("detail-desc");
    if (descCard && descText) {
        if (todo.description) {
            descText.textContent = todo.description;
            descCard.hidden = false;
        } else {
            descCard.hidden = true;
        }
    }
    
    // Tags Card
    const tagsCard = document.getElementById("detail-tags-card");
    const tagsContainer = document.getElementById("detail-tags");
    if (tagsCard && tagsContainer) {
        if (todo.tags && todo.tags.length > 0) {
            tagsContainer.innerHTML = todo.tags.map(tag => `<span class="tag-badge">${escapeHTML(String(tag))}</span>`).join("");
            tagsCard.hidden = false;
        } else {
            tagsCard.hidden = true;
        }
    }
    
    // Sub-tasks Card
    const detailSubtasksCard = document.getElementById("detail-subtasks-card");
    const detailSubtasksContainer = document.getElementById("detail-subtasks-container");
    if (detailSubtasksCard && detailSubtasksContainer) {
        if (todo.sub_tasks && todo.sub_tasks.length > 0) {
            detailSubtasksContainer.innerHTML = "";
            todo.sub_tasks.forEach((st) => {
                const isChecked = st.status === "done";
                const stEl = document.createElement("button");
                stEl.type = "button";
                stEl.className = `detail-subtask-item ${isChecked ? "done" : ""}`;
                stEl.setAttribute("aria-pressed", String(isChecked));
                stEl.setAttribute("aria-label", `${isChecked ? "Tandai belum selesai" : "Tandai selesai"}: ${st.title}`);
                stEl.innerHTML = `
                    <span class="subtask-checkbox ${isChecked ? "checked" : ""}" aria-hidden="true"></span>
                    <span class="subtask-title">${escapeHTML(st.title)}</span>
                `;
                stEl.addEventListener("click", () => toggleSubTaskStatus(todo.id, st.id));
                detailSubtasksContainer.appendChild(stEl);
            });
            detailSubtasksCard.hidden = false;
        } else {
            detailSubtasksCard.hidden = true;
        }
    }
    if (window.innerWidth < 768) detailBackBtn.focus({ preventScroll: true });
}

function clearSelection() {
    const wasMobileDetailOpen = detailPanel.classList.contains("mobile-open");
    const previousId = selectedTodoId;
    selectedTodoId = null;
    document.querySelectorAll(".todo-item").forEach(item => item.classList.remove("active-todo"));
    dashboardContent.classList.remove("has-selection");
    detailPanel.classList.remove("mobile-open");
    detailPanel.setAttribute("aria-hidden", "true");
    detailPanel.querySelector(".detail-empty-state").hidden = false;
    detailPanel.querySelector(".detail-content").hidden = true;
    detailEmptyHint.hidden = !(hasAnyTodos && window.innerWidth >= 768);
    if (wasMobileDetailOpen) {
        const previousCard = Array.from(document.querySelectorAll(".todo-item"))
            .find(item => item.dataset.id === previousId);
        if (previousCard) previousCard.focus({ preventScroll: true });
    }
}

// Detail Panel Button Event Listeners
document.getElementById("detail-edit-btn").addEventListener("click", () => {
    if (selectedTodoId) openEditModal(selectedTodoId);
});

document.getElementById("detail-delete-btn").addEventListener("click", () => {
    if (selectedTodoId) {
        deleteTodo(selectedTodoId);
    }
});

// ── SELECTION & BULK ACTIONS ──

// Toggle selection mode on/off
function toggleSelectionMode() {
    isSelectionMode = !isSelectionMode;
    selectedTodos.clear();

    const selectModeLabel = document.getElementById("select-mode-label");
    selectModeBtn.setAttribute("aria-pressed", String(isSelectionMode));
    if (isSelectionMode) {
        selectModeLabel.textContent = "Batal pilih";
        bulkActionBar.classList.add("active");
        clearSelection();
    } else {
        selectModeLabel.textContent = "Pilih beberapa";
        bulkActionBar.classList.remove("active");
    }

    updateBulkBarUI();
    renderTodoList(currentTodos);
}

// Toggle a single todo selection state
function toggleTodoSelection(id) {
    if (selectedTodos.has(id)) {
        selectedTodos.delete(id);
    } else {
        selectedTodos.add(id);
    }
    
    const todoEl = document.querySelector(`.todo-item[data-id="${id}"]`);
    if (todoEl) {
        const isSelected = selectedTodos.has(id);
        todoEl.classList.toggle("selected-item", isSelected);
        const checkbox = todoEl.querySelector(".checkbox-circle");
        const todo = currentTodos.find(item => item.id === id);
        checkbox.classList.toggle("selected", isSelected);
        checkbox.setAttribute("aria-pressed", String(isSelected));
        checkbox.setAttribute("aria-label", `${isSelected ? "Batalkan pilihan" : "Pilih"}: ${todo.title}`);
        checkbox.title = checkbox.getAttribute("aria-label");
    }

    updateBulkBarUI();
}

// Update the bulk bar count and state
function updateBulkBarUI() {
    const count = selectedTodos.size;
    bulkSelectCount.textContent = `${count} Terpilih`;
    const disabled = count === 0;
    bulkCompleteBtn.disabled = disabled;
    bulkDeleteBtn.disabled = disabled;
}

// Handler for checkbox click (intercepts individual action if in selection mode)
function handleCheckboxClick(e, id, status) {
    e.stopPropagation();
    if (isSelectionMode) {
        toggleTodoSelection(id);
    } else {
        toggleComplete(id, status);
    }
}

// Trigger bulk complete
async function triggerBulkComplete() {
    if (selectedTodos.size === 0) return;
    
    try {
        const res = await request("/todos/bulk-complete", {
            method: "POST",
            body: JSON.stringify({ ids: Array.from(selectedTodos) })
        });
        
        if (res && res.success) {
            showToast(res.message || "Tugas berhasil diselesaikan secara massal!");
            toggleSelectionMode(); // Exit selection mode
            loadTodos();
        }
    } catch (err) {
        // handled
    }
}

// Trigger bulk delete
async function triggerBulkDelete() {
    if (selectedTodos.size === 0) return;
    if (!confirm(`Apakah Anda yakin ingin menghapus ${selectedTodos.size} tugas ini?`)) return;
    
    try {
        const res = await request("/todos/bulk-delete", {
            method: "POST",
            body: JSON.stringify({ ids: Array.from(selectedTodos) })
        });
        
        if (res && res.success) {
            showToast(res.message || "Tugas berhasil dihapus secara massal!");
            toggleSelectionMode(); // Exit selection mode
            loadTodos();
        }
    } catch (err) {
        // handled
    }
}

// Register Selection event listeners
selectModeBtn.addEventListener("click", toggleSelectionMode);
bulkCompleteBtn.addEventListener("click", triggerBulkComplete);
bulkDeleteBtn.addEventListener("click", triggerBulkDelete);

// ── THEME HANDLER ──
function setProfileMenuOpen(isOpen) {
    profileMenu.hidden = !isOpen;
    profileMenuTrigger.setAttribute("aria-expanded", String(isOpen));
}

function setSidebarOpen(isOpen) {
    const shouldOpen = isOpen && window.innerWidth < 768;
    const wasOpen = dashboardContent.classList.contains("drawer-open");
    dashboardContent.classList.toggle("drawer-open", shouldOpen);
    sidebarToggleBtn.setAttribute("aria-expanded", String(shouldOpen));
    sidebarBackdrop.tabIndex = -1;
    if (shouldOpen) {
        sidebarCloseBtn.focus({ preventScroll: true });
    } else if (wasOpen && window.innerWidth < 768) {
        sidebarToggleBtn.focus({ preventScroll: true });
    }
}

function initTheme() {
    const savedTheme = localStorage.getItem("todo_theme");
    const systemPrefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    const initialTheme = savedTheme || (systemPrefersDark ? "dark" : "light");
    
    applyTheme(initialTheme);
}

function applyTheme(theme) {
    const isDark = theme === "dark";
    document.documentElement.classList.toggle("dark-theme", isDark);
    document.body.classList.toggle("dark-theme", isDark);
    themeActionLabel.textContent = isDark ? "Ganti ke tema terang" : "Ganti ke tema gelap";
    themeToggleBtn.title = isDark ? "Ganti ke tema terang" : "Ganti ke tema gelap";
    themeToggleBtn.setAttribute("aria-label", themeToggleBtn.title);
}

function toggleTheme() {
    const isDark = document.documentElement.classList.contains("dark-theme");
    const newTheme = isDark ? "light" : "dark";
    localStorage.setItem("todo_theme", newTheme);
    applyTheme(newTheme);
}

if (themeToggleBtn) {
    themeToggleBtn.addEventListener("click", () => {
        toggleTheme();
        setProfileMenuOpen(false);
    });
}

document.addEventListener("keydown", event => {
    const target = event.target;
    const isEditing = target instanceof HTMLElement
        && (target.matches("input, textarea, select") || target.isContentEditable);

    if (todoModal.classList.contains("active") && event.key === "Tab") {
        const focusable = Array.from(todoModal.querySelectorAll(
            'button:not([disabled]), input:not([disabled]):not([type="hidden"]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
        )).filter(element => !element.closest("[hidden]"));
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
        return;
    }

    if (event.key === "Tab" && dashboardContent.classList.contains("drawer-open")) {
        const focusable = Array.from(dashboardContent.querySelectorAll(
            '.dashboard-sidebar button:not([disabled]), .dashboard-sidebar select:not([disabled])'
        ));
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
        return;
    }

    if (event.key === "Tab" && detailPanel.classList.contains("mobile-open")) {
        const focusable = Array.from(detailPanel.querySelectorAll(
            'button:not([disabled]), [tabindex="0"]'
        )).filter(element => !element.closest("[hidden]"));
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
        return;
    }

    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        searchInput.focus();
        searchInput.select();
        return;
    }

    if (event.key === "Escape") {
        if (todoModal.classList.contains("active")) {
            closeModal();
        } else if (profileMenu && !profileMenu.hidden) {
            setProfileMenuOpen(false);
            profileMenuTrigger.focus();
        } else if (dashboardContent.classList.contains("drawer-open")) {
            setSidebarOpen(false);
        } else if (detailPanel.classList.contains("mobile-open")) {
            clearSelection();
        }
        return;
    }

    if (!isEditing && !event.ctrlKey && !event.metaKey && !event.altKey && event.key.toLowerCase() === "n") {
        if (dashboardScreen.classList.contains("active") && !todoModal.classList.contains("active")) {
            event.preventDefault();
            openCreateModal();
        }
    }

    if (profileMenu && !profileMenu.hidden && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
        const items = Array.from(profileMenu.querySelectorAll('[role="menuitem"]'));
        const index = items.indexOf(document.activeElement);
        const nextIndex = event.key === "ArrowDown"
            ? (index + 1) % items.length
            : (index - 1 + items.length) % items.length;
        event.preventDefault();
        items[nextIndex < 0 ? 0 : nextIndex].focus();
    }
});

window.addEventListener("resize", () => {
    if (selectedTodoId) {
        detailPanel.classList.toggle("mobile-open", window.innerWidth < 768);
    }
    if (window.innerWidth >= 768) {
        dashboardContent.classList.remove("drawer-open");
        sidebarToggleBtn.setAttribute("aria-expanded", "false");
    } else {
        detailEmptyHint.hidden = true;
    }
});

// ── SCROLL OBSERVER ──
function initScrollObserver() {
    if (!scrollSentinel) return;
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting && hasMoreTodos && !isLoadingMore) {
                loadMoreTodos();
            }
        });
    }, {
        root: null, // browser viewport
        rootMargin: "0px 0px 100px 0px", // pre-fetch
        threshold: 0.1
    });
    
    observer.observe(scrollSentinel);
}

// ── INIT ──
initTheme();
initScrollObserver();
checkAuth();
