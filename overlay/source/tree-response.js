    processResponse : function(response, node, callback){
        try {
            var o = Ext.decode(response.responseText);
            if(!Ext.isArray(o)){
                throw new SyntaxError('TreeLoader response must be a JSON array');
            }
            // Validate and construct all immediate children before changing the tree.
            var children = [];
            for(var i = 0; i < o.length; i++){
                children.push(this.createNode(o[i]));
            }
            node.beginUpdate();
            try {
                for(var i = 0; i < children.length; i++){
                    if(children[i]){
                        node.appendChild(children[i]);
                    }
                }
            } finally {
                node.endUpdate();
            }
        }catch(e){
            this.handleFailure(response);
            return false;
        }
        if(typeof callback == 'function'){
            callback(this, node);
        }
        return true;
    },
