// Export utility functions for CSV and JSON

export function exportToCSV(data: Record<string, unknown>[], filename: string) {
    if (data.length === 0) {
        alert('No data to export');
        return;
    }

    // Get headers from first object
    const headers = Object.keys(data[0]);

    // Create CSV content
    const csvContent = [
        headers.join(','), // Header row
        ...data.map(row =>
            headers.map(header => {
                const cell = row[header];
                // Handle commas, quotes, and newlines in data
                if (cell === null || cell === undefined) return '';
                const cellStr = String(cell).replace(/"/g, '""');
                return cellStr.includes(',') || cellStr.includes('"') || cellStr.includes('\n')
                    ? `"${cellStr}"`
                    : cellStr;
            }).join(',')
        )
    ].join('\n');

    // Create and trigger download
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);

    link.setAttribute('href', url);
    link.setAttribute('download', `${filename}_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.visibility = 'hidden';

    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

export function exportToJSON(data: Record<string, unknown>[], filename: string) {
    if (data.length === 0) {
        alert('No data to export');
        return;
    }

    const jsonContent = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonContent], { type: 'application/json;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);

    link.setAttribute('href', url);
    link.setAttribute('download', `${filename}_${new Date().toISOString().split('T')[0]}.json`);
    link.style.visibility = 'hidden';

    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

export function prepareDataForExport(data: Record<string, unknown>[], fields?: string[]) {
    if (!fields) return data;

    return data.map(item => {
        const exported: Record<string, unknown> = {};
        fields.forEach(field => {
            if (field.includes('.')) {
                const parts = field.split('.');
                let value: unknown = item;
                for (const part of parts) {
                    value = (value as Record<string, unknown>)?.[part];
                }
                exported[field] = value;
            } else {
                exported[field] = item[field];
            }
        });
        return exported;
    });
}
