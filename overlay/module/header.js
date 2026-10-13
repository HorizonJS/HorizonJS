// HorizonJS browser distribution. See LICENSE.txt for upstream terms.
if (typeof window === 'undefined' || typeof document === 'undefined') {
    throw new Error('HorizonJS requires a browser DOM. Import it on the client.');
}
if (window.Ext) {
    throw new Error('HorizonJS is already loaded. Use one module or script distribution per page.');
}
var Ext;
