    getJsonAccessor: function(expr) {
        if(typeof expr == 'number' && isFinite(expr) && expr >= 0 && Math.floor(expr) == expr){
            expr = String(expr);
        }
        if(typeof expr != 'string' || !expr.length){
            throw new SyntaxError('Invalid JSON property path');
        }
        var parts = [], rest = expr, match, key;
        var first = /^[\w$-]+/;
        var dotted = /^\.([\w$-]+)/;
        var bracket = /^\[\s*(?:(\d+)|"((?:\\.|[^"\\\x00-\x1f])*)"|'((?:\\.|[^'\\\x00-\x1f])*)')\s*\]/;
        match = rest.match(first);
        if(match){
            parts.push(match[0]);
            rest = rest.substr(match[0].length);
        }
        while(rest){
            match = rest.match(dotted);
            if(match && parts.length){
                key = match[1];
            }else{
                match = rest.match(bracket);
                if(!match){
                    throw new SyntaxError('Invalid JSON property path: ' + expr);
                }
                if(match[1] !== undefined){
                    key = match[1];
                }else if(match[2] !== undefined){
                    key = JSON.parse('"' + match[2] + '"');
                }else{
                    // Decode a single-quoted property key without evaluating it.
                    var raw = match[3], decoded = '', escapes = {
                        "'": "'", '"': '"', '\\': '\\', '/': '/',
                        b: '\b', f: '\f', n: '\n', r: '\r', t: '\t'
                    };
                    for(var i = 0; i < raw.length; i++){
                        var c = raw.charAt(i);
                        if(c == '\\'){
                            c = raw.charAt(++i);
                            if(c == 'u' && /^[0-9a-fA-F]{4}$/.test(raw.substr(i+1, 4))){
                                decoded += String.fromCharCode(parseInt(raw.substr(i+1, 4), 16));
                                i += 4;
                            }else if(Object.prototype.hasOwnProperty.call(escapes, c)){
                                decoded += escapes[c];
                            }else{
                                throw new SyntaxError('Invalid escape in JSON property path');
                            }
                        }else{
                            decoded += c;
                        }
                    }
                    key = decoded;
                }
            }
            parts.push(key);
            rest = rest.substr(match[0].length);
        }
        for(var i = 0; i < parts.length; i++){
            if(parts[i] == '__proto__' || parts[i] == 'prototype' || parts[i] == 'constructor'){
                throw new SyntaxError('Unsafe JSON property path: ' + expr);
            }
        }
        return function(obj){
            for(var i = 0; i < parts.length; i++){
                if(obj === null || obj === undefined || !Object.prototype.hasOwnProperty.call(obj, parts[i])){
                    return undefined;
                }
                obj = obj[parts[i]];
            }
            return obj;
        };
    },
