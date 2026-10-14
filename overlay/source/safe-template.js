Ext.Template.from = function(el, config){
    el = Ext.getDom(el);
    return new Ext.Template(el.value || el.innerHTML, config || '');
};

(function(){
    var own = Object.prototype.hasOwnProperty;
    var tags = Object.create(null);
    var attributes = Object.create(null);
    var names = 'div span p b strong i em u s small sub sup br hr pre code blockquote ul ol li dl dt dd h1 h2 h3 h4 h5 h6 table caption colgroup col thead tbody tfoot tr th td section article main aside header footer nav'.split(' ');
    for(var i = 0; i < names.length; i++){
        tags[names[i]] = true;
    }
    names = 'id class title role lang dir colspan rowspan scope'.split(' ');
    for(var i = 0; i < names.length; i++){
        attributes[names[i]] = true;
    }

    function walk(root, visit){
        for(var node = root.firstChild; node; node = node.nextSibling){
            visit(node);
            if(node.firstChild){
                walk(node, visit);
            }
        }
    }

    function parts(text){
        var result = [], re = /\{\{|\}\}|\{([^{}]+)\}|[{}]/g, match, offset = 0;
        while((match = re.exec(text))){
            result.push(text.substring(offset, match.index));
            if(match[0] == '{{' || match[0] == '}}'){
                result.push(match[0].charAt(0));
            }else{
                var path = match[1];
                if(!path || !/^[\w$-]+(?:\.[\w$-]+)*$/.test(path)){
                    throw new SyntaxError('SafeTemplate accepts data paths, not expressions or formatters');
                }
                path = path.split('.');
                for(var i = 0; i < path.length; i++){
                    if(path[i] == '__proto__' || path[i] == 'constructor' || path[i] == 'prototype'){
                        throw new SyntaxError('Unsafe SafeTemplate data path');
                    }
                }
                result.push({path:path});
            }
            offset = re.lastIndex;
        }
        result.push(text.substr(offset));
        return result;
    }

    function value(data, path){
        for(var i = 0; i < path.length; i++){
            if(data === null || data === undefined || !own.call(Object(data), path[i])){
                return '';
            }
            var descriptor = Object.getOwnPropertyDescriptor(Object(data), path[i]);
            if(!own.call(descriptor, 'value')){
                throw new TypeError('SafeTemplate does not invoke data getters');
            }
            data = descriptor.value;
        }
        if(data === null || data === undefined){
            return '';
        }
        if(typeof data != 'string' && typeof data != 'number' && typeof data != 'boolean'){
            throw new TypeError('SafeTemplate values must be text, numbers, booleans, or null');
        }
        return String(data);
    }

    function container(node){
        if(!node || node.nodeType != 1 || node.namespaceURI != 'http://www.w3.org/1999/xhtml' || (!tags[node.tagName.toLowerCase()] && node.tagName.toLowerCase() != 'body')){
            throw new TypeError('SafeTemplate requires an HTML content container');
        }
        return node;
    }

    /**
     * @class Ext.SafeTemplate
     * A text-oriented template with no executable expressions. Bindings such as
     * {name}, {person.name}, and {items.0.name} read own data properties and create
     * text nodes. Static markup uses a restricted HTML subset; attribute bindings,
     * URLs, styles, event handlers, active elements, and formatters are rejected.
     * Use doubled braces to display literal braces. Legacy Template and XTemplate
     * remain available for trusted application-authored executable definitions.
     * @constructor
     * @param {String} html The static HTML and text placeholders
     */
    Ext.SafeTemplate = function(html){
        this.set(html);
    };

    Ext.SafeTemplate.prototype = {
        /**
         * Validates and sets a template without executing code. Invalid input
         * leaves the previous template intact.
         * @param {String} html
         * @return {Ext.SafeTemplate} this
         */
        set : function(html){
            if(typeof html != 'string'){
                throw new TypeError('SafeTemplate requires a string');
            }
            var holder = document.createElement('template');
            holder.innerHTML = html;
            var bindings = [];
            walk(holder.content, function(node){
                if(node.nodeType == 1){
                    if(node.namespaceURI != 'http://www.w3.org/1999/xhtml' || !tags[node.tagName.toLowerCase()]){
                        throw new SyntaxError('Unsupported SafeTemplate element: ' + node.tagName);
                    }
                    for(var i = 0; i < node.attributes.length; i++){
                        var attribute = node.attributes[i], name = attribute.name.toLowerCase();
                        if((!attributes[name] && !/^(?:aria|data)-[a-z0-9-]+$/.test(name)) || /[{}]/.test(attribute.value)){
                            throw new SyntaxError('Unsupported SafeTemplate attribute or attribute binding: ' + name);
                        }
                    }
                }else if(node.nodeType == 3){
                    bindings.push(parts(node.nodeValue));
                }else if(node.nodeType != 8 || /[{}]/.test(node.nodeValue)){
                    throw new SyntaxError('Unsupported SafeTemplate node or comment binding');
                }
            });
            this.html = html;
            this.fragment = holder.content;
            this.bindings = bindings;
            return this;
        },

        /**
         * Validates the current HTML again. No eval or Function compilation is used.
         * @return {Ext.SafeTemplate} this
         */
        compile : function(){
            return this.set(this.html);
        },

        /**
         * Creates a new fragment containing static markup and bound text nodes.
         * Accessors and object-to-string conversion are not invoked.
         * @param {Object/Array} values Data used for text bindings
         * @return {DocumentFragment} A fresh fragment
         */
        render : function(values){
            var fragment = this.fragment.cloneNode(true), bindings = this.bindings, index = 0;
            walk(fragment, function(node){
                if(node.nodeType == 3){
                    var tokens = bindings[index++], text = '';
                    for(var i = 0; i < tokens.length; i++){
                        text += typeof tokens[i] == 'string' ? tokens[i] : value(values, tokens[i].path);
                    }
                    node.nodeValue = text;
                }
            });
            return fragment;
        },

        /**
         * Returns HTML with data escaped by text-node serialization.
         * @param {Object/Array} values
         * @return {String} HTML suitable for an HTML content context
         */
        apply : function(values){
            var holder = document.createElement('div');
            holder.appendChild(this.render(values));
            return holder.innerHTML;
        },

        /** @private */
        doInsert : function(where, el, values, returnElement){
            el = container(Ext.getDom(el));
            var parent = el;
            if(where == 'beforeBegin' || where == 'afterEnd'){
                parent = container(el.parentNode);
            }
            var fragment = this.render(values), first = fragment.firstChild;
            if(where == 'beforeBegin'){
                parent.insertBefore(fragment, el);
            }else if(where == 'afterEnd'){
                parent.insertBefore(fragment, el.nextSibling);
            }else if(where == 'afterBegin'){
                el.insertBefore(fragment, el.firstChild);
            }else if(where == 'beforeEnd'){
                el.appendChild(fragment);
            }else if(where == 'overwrite'){
                while(el.firstChild){
                    el.removeChild(el.firstChild);
                }
                el.appendChild(fragment);
            }else{
                throw new Error('Unknown SafeTemplate insertion position');
            }
            return returnElement && first && first.nodeType == 1 ? Ext.get(first, true) : first;
        },

        /**
         * Inserts bound nodes as the first children of a supported HTML container.
         * @param {String/HTMLElement/Ext.Element} el The destination
         * @param {Object/Array} values Text-binding data
         * @param {Boolean} returnElement True to wrap an element result
         * @return {Node/Ext.Element} The first inserted node, its wrapper, or null for empty output
         */
        insertFirst : function(el, values, returnElement){
            return this.doInsert('afterBegin', el, values, returnElement);
        },
        /**
         * Inserts bound nodes before the destination in its supported HTML parent.
         * @param {String/HTMLElement/Ext.Element} el The destination
         * @param {Object/Array} values Text-binding data
         * @param {Boolean} returnElement True to wrap an element result
         * @return {Node/Ext.Element} The first inserted node, its wrapper, or null for empty output
         */
        insertBefore : function(el, values, returnElement){
            return this.doInsert('beforeBegin', el, values, returnElement);
        },
        /**
         * Inserts bound nodes after the destination in its supported HTML parent.
         * @param {String/HTMLElement/Ext.Element} el The destination
         * @param {Object/Array} values Text-binding data
         * @param {Boolean} returnElement True to wrap an element result
         * @return {Node/Ext.Element} The first inserted node, its wrapper, or null for empty output
         */
        insertAfter : function(el, values, returnElement){
            return this.doInsert('afterEnd', el, values, returnElement);
        },
        /**
         * Appends bound nodes to a supported HTML container using DOM operations.
         * @param {String/HTMLElement/Ext.Element} el The destination
         * @param {Object/Array} values Text-binding data
         * @param {Boolean} returnElement True to wrap an element result
         * @return {Node/Ext.Element} The first inserted node, its wrapper, or null for empty output
         */
        append : function(el, values, returnElement){
            return this.doInsert('beforeEnd', el, values, returnElement);
        },
        /**
         * Replaces container children after rendering succeeds. Invalid data
         * leaves existing content intact. Active content containers are rejected.
         * @param {String/HTMLElement/Ext.Element} el The destination
         * @param {Object/Array} values Text-binding data
         * @param {Boolean} returnElement True to wrap an element result
         * @return {Node/Ext.Element} The first inserted node, its wrapper, or null for empty output
         */
        overwrite : function(el, values, returnElement){
            return this.doInsert('overwrite', el, values, returnElement);
        }
    };
    /**
     * Alias for {@link #apply}.
     * @method
     */
    Ext.SafeTemplate.prototype.applyTemplate = Ext.SafeTemplate.prototype.apply;
})();
